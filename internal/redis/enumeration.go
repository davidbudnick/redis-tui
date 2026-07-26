package redis

import (
	"container/heap"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/davidbudnick/redis-tui/internal/decode"
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// GetTotalKeys returns the total number of keys in the current database
func (c *Client) GetTotalKeys() int64 {
	count, err := c.cmdable().DBSize(c.ctx).Result()
	if err != nil {
		return 0
	}
	return count
}

// ScanKeys scans keys matching a pattern
func (c *Client) ScanKeys(pattern string, cursor uint64, count int64) ([]types.RedisKey, uint64, error) {
	if pattern == "" {
		pattern = "*"
	}

	c.mu.RLock()
	includeTypes := c.includeTypes
	isCluster := c.isCluster
	client := c.client
	ctx := c.ctx
	c.mu.RUnlock()

	var keys []string
	var nextCursor uint64
	var err error

	if isCluster {
		// In cluster mode, scan all masters to get keys from every shard
		keys, err = c.scanAll(pattern, count)
		nextCursor = 0
	} else {
		keys, nextCursor, err = client.Scan(ctx, cursor, pattern, count).Result()
	}
	if err != nil {
		return nil, 0, err
	}

	if len(keys) == 0 {
		return []types.RedisKey{}, nextCursor, nil
	}

	// Use pipeline to batch TTL (and optionally TYPE) calls
	pipe := c.pipeline()
	var typeCmds []*redis.StatusCmd
	ttlCmds := make([]*redis.DurationCmd, len(keys))

	if includeTypes {
		typeCmds = make([]*redis.StatusCmd, len(keys))
		for i, key := range keys {
			typeCmds[i] = pipe.Type(ctx, key)
			ttlCmds[i] = pipe.TTL(ctx, key)
		}
	} else {
		for i, key := range keys {
			ttlCmds[i] = pipe.TTL(ctx, key)
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, 0, err
	}

	result := make([]types.RedisKey, len(keys))
	for i, key := range keys {
		var keyType string
		if includeTypes && typeCmds != nil {
			keyType, _ = typeCmds[i].Result()
		}
		ttl, _ := ttlCmds[i].Result()
		result[i] = types.RedisKey{
			Key:  key,
			Type: types.KeyType(keyType),
			TTL:  ttl,
		}
	}

	// Detect subtypes (HLL, Bitmap for strings; Geo for zsets)
	if includeTypes {
		result = c.detectStringSubtypes(result)
		result = c.detectZSetSubtypes(result)
	}

	return result, nextCursor, nil
}

const (
	// scanBatchDefault is the SCAN COUNT hint for key enumeration.
	scanBatchDefault int64 = 100
	// scanBatchPrefixes is the SCAN COUNT hint used when building the prefix tree.
	scanBatchPrefixes int64 = 500
	// maxRegexLen is the maximum allowed length for user-provided regex patterns.
	maxRegexLen = 1024
	// subtypeProbeBytes is how many leading bytes of a string value are fetched
	// (via GETRANGE) to detect HLL/protobuf/bitmap subtypes without transferring
	// the full value on every scan page.
	subtypeProbeBytes = 256
	// subtypeFullFetchMax is the largest string we will fully GET during scan
	// classification when a short probe looks binary but is not yet classifiable
	// (e.g. s2-compressed protobuf needs the complete blob to decode).
	subtypeFullFetchMax int64 = 512 * 1024
	// defaultSearchMaxKeys is used when callers pass maxKeys <= 0.
	defaultSearchMaxKeys = 100
	// searchMaxStringBytes caps string bytes inspected during value search
	// (GETRANGE prefix). Matches outside this window may be missed.
	searchMaxStringBytes int64 = previewMaxStringBytes
	// searchMaxItems caps collection entries inspected during value search.
	// Matches beyond this prefix window may be missed.
	searchMaxItems = previewMaxItems
)

// ScanKeysWithRegex scans keys using regex pattern with early termination.
// Uses incremental SCAN to avoid loading the full keyspace into memory.
func (c *Client) ScanKeysWithRegex(regexPattern string, maxKeys int) ([]types.RedisKey, error) {
	if len(regexPattern) > maxRegexLen {
		return nil, errInvalidRegex(fmt.Errorf("pattern exceeds maximum length of %d characters", maxRegexLen))
	}
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, errInvalidRegex(err)
	}

	matchingKeys := make([]string, 0, maxKeys)
	scanErr := c.scanEach("*", scanBatchDefault, func(keys []string) bool {
		for _, key := range keys {
			if re.MatchString(key) {
				matchingKeys = append(matchingKeys, key)
				if len(matchingKeys) >= maxKeys {
					return false
				}
			}
		}
		return true
	})
	if scanErr != nil {
		return nil, scanErr
	}

	if len(matchingKeys) == 0 {
		return []types.RedisKey{}, nil
	}

	// Use pipeline to batch Type and TTL calls
	pipe := c.pipeline()
	typeCmds := make([]*redis.StatusCmd, len(matchingKeys))
	ttlCmds := make([]*redis.DurationCmd, len(matchingKeys))

	for i, key := range matchingKeys {
		typeCmds[i] = pipe.Type(c.ctx, key)
		ttlCmds[i] = pipe.TTL(c.ctx, key)
	}

	_, _ = pipe.Exec(c.ctx)

	result := make([]types.RedisKey, len(matchingKeys))
	for i, key := range matchingKeys {
		keyType, _ := typeCmds[i].Result()
		ttl, _ := ttlCmds[i].Result()
		result[i] = types.RedisKey{
			Key:  key,
			Type: types.KeyType(keyType),
			TTL:  ttl,
		}
	}

	return result, nil
}

// FuzzySearchKeys performs fuzzy matching on key names.
// Scans the full keyspace for correct global top-N, keeping only O(maxKeys) scored candidates.
func (c *Client) FuzzySearchKeys(searchTerm string, maxKeys int) ([]types.RedisKey, error) {
	if maxKeys <= 0 {
		maxKeys = defaultSearchMaxKeys
	}
	searchLower := strings.ToLower(searchTerm)

	top := make(scoreMinHeap, 0, maxKeys)
	err := c.scanEach("*", scanBatchDefault, func(keys []string) bool {
		for _, key := range keys {
			score := fuzzyScore(strings.ToLower(key), searchLower)
			if score > 0 {
				top.consider(scoredKey{key: key, score: score}, maxKeys)
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	if top.Len() == 0 {
		return []types.RedisKey{}, nil
	}

	scoredKeys := make([]scoredKey, top.Len())
	for i := len(scoredKeys) - 1; i >= 0; i-- {
		scoredKeys[i] = heap.Pop(&top).(scoredKey)
	}
	sort.SliceStable(scoredKeys, func(i, j int) bool {
		return scoredKeys[i].score > scoredKeys[j].score
	})

	pipe := c.pipeline()
	typeCmds := make([]*redis.StatusCmd, len(scoredKeys))
	ttlCmds := make([]*redis.DurationCmd, len(scoredKeys))
	for i, sk := range scoredKeys {
		typeCmds[i] = pipe.Type(c.ctx, sk.key)
		ttlCmds[i] = pipe.TTL(c.ctx, sk.key)
	}
	_, _ = pipe.Exec(c.ctx)

	result := make([]types.RedisKey, len(scoredKeys))
	for i, sk := range scoredKeys {
		keyType, _ := typeCmds[i].Result()
		ttl, _ := ttlCmds[i].Result()
		result[i] = types.RedisKey{
			Key:  sk.key,
			Type: types.KeyType(keyType),
			TTL:  ttl,
		}
	}
	return result, nil
}

type scoredKey struct {
	key   string
	score int
}

type scoreMinHeap []scoredKey

func (h scoreMinHeap) Len() int           { return len(h) }
func (h scoreMinHeap) Less(i, j int) bool { return h[i].score < h[j].score }
func (h scoreMinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *scoreMinHeap) Push(x any) { *h = append(*h, x.(scoredKey)) }

func (h *scoreMinHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func (h *scoreMinHeap) consider(sk scoredKey, maxKeys int) {
	if maxKeys <= 0 {
		return
	}
	if h.Len() < maxKeys {
		heap.Push(h, sk)
		return
	}
	if sk.score > (*h)[0].score {
		(*h)[0] = sk
		heap.Fix(h, 0)
	}
}

func fuzzyScore(str, pattern string) int {
	if strings.Contains(str, pattern) {
		return 100 + (len(str) - len(pattern))
	}

	score := 0
	patternIdx := 0

	for i := 0; i < len(str) && patternIdx < len(pattern); i++ {
		if str[i] == pattern[patternIdx] {
			score += 10
			if i > 0 && (str[i-1] == ':' || str[i-1] == '_' || str[i-1] == '-') {
				score += 5
			}
			patternIdx++
		}
	}

	if patternIdx == len(pattern) {
		return score
	}
	return 0
}

// SearchByValue searches for keys containing a value.
// Streams SCAN batches, bounds value fetches to a prefix window, and stops at maxKeys.
// Matches outside the inspected prefix (string/collection window) may be missed.
func (c *Client) SearchByValue(pattern string, valueSearch string, maxKeys int) ([]types.RedisKey, error) {
	if maxKeys <= 0 {
		maxKeys = defaultSearchMaxKeys
	}
	if pattern == "" {
		pattern = "*"
	}

	type match struct {
		key     string
		keyType string
	}
	matches := make([]match, 0, maxKeys)

	err := c.scanEach(pattern, scanBatchDefault, func(keys []string) bool {
		typePipe := c.pipeline()
		typeCmds := make([]*redis.StatusCmd, len(keys))
		for j, key := range keys {
			typeCmds[j] = typePipe.Type(c.ctx, key)
		}
		_, _ = typePipe.Exec(c.ctx)

		keyTypes := make([]string, len(keys))
		for j := range keys {
			keyTypes[j], _ = typeCmds[j].Result()
		}

		valuePipe := c.pipeline()
		type valueCmd struct {
			idx     int
			keyType string
			strCmd  *redis.StringCmd
			hashCmd *redis.ScanCmd
			listCmd *redis.StringSliceCmd
			setCmd  *redis.ScanCmd
			jsonCmd *redis.Cmd
		}
		valueCmds := make([]valueCmd, 0, len(keys))

		for j, key := range keys {
			kt := keyTypes[j]
			vc := valueCmd{idx: j, keyType: kt}
			switch kt {
			case "string":
				vc.strCmd = valuePipe.GetRange(c.ctx, key, 0, searchMaxStringBytes-1)
			case "hash":
				vc.hashCmd = valuePipe.HScan(c.ctx, key, 0, "", int64(searchMaxItems))
			case "list":
				vc.listCmd = valuePipe.LRange(c.ctx, key, 0, int64(searchMaxItems-1))
			case "set":
				vc.setCmd = valuePipe.SScan(c.ctx, key, 0, "", int64(searchMaxItems))
			case "ReJSON-RL":
				vc.jsonCmd = valuePipe.Do(c.ctx, "JSON.GET", key, "$")
			default:
				continue
			}
			valueCmds = append(valueCmds, vc)
		}
		if len(valueCmds) > 0 {
			_, _ = valuePipe.Exec(c.ctx)
		}

		for _, vc := range valueCmds {
			if valueCmdMatches(vc.keyType, valueSearch, vc.strCmd, vc.hashCmd, vc.listCmd, vc.setCmd, vc.jsonCmd) {
				matches = append(matches, match{key: keys[vc.idx], keyType: keyTypes[vc.idx]})
				if len(matches) >= maxKeys {
					return false
				}
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return []types.RedisKey{}, nil
	}

	ttlPipe := c.pipeline()
	ttlCmds := make([]*redis.DurationCmd, len(matches))
	for j, m := range matches {
		ttlCmds[j] = ttlPipe.TTL(c.ctx, m.key)
	}
	_, _ = ttlPipe.Exec(c.ctx)

	result := make([]types.RedisKey, len(matches))
	for j, m := range matches {
		ttl, _ := ttlCmds[j].Result()
		result[j] = types.RedisKey{
			Key:  m.key,
			Type: types.KeyType(m.keyType),
			TTL:  ttl,
		}
	}
	return result, nil
}

func valueCmdMatches(
	keyType, valueSearch string,
	strCmd *redis.StringCmd,
	hashCmd *redis.ScanCmd,
	listCmd *redis.StringSliceCmd,
	setCmd *redis.ScanCmd,
	jsonCmd *redis.Cmd,
) bool {
	switch keyType {
	case "string":
		if strCmd == nil {
			return false
		}
		val, _ := strCmd.Result()
		return strings.Contains(val, valueSearch)
	case "hash":
		if hashCmd == nil {
			return false
		}
		batch, _, _ := hashCmd.Result()
		for i := 0; i+1 < len(batch); i += 2 {
			if strings.Contains(batch[i+1], valueSearch) {
				return true
			}
		}
	case "list":
		if listCmd == nil {
			return false
		}
		vals, _ := listCmd.Result()
		for _, v := range vals {
			if strings.Contains(v, valueSearch) {
				return true
			}
		}
	case "set":
		if setCmd == nil {
			return false
		}
		vals, _, _ := setCmd.Result()
		for _, v := range vals {
			if strings.Contains(v, valueSearch) {
				return true
			}
		}
	case "ReJSON-RL":
		if jsonCmd == nil {
			return false
		}
		val, err := jsonCmd.Text()
		if err != nil {
			return false
		}
		return strings.Contains(val, valueSearch)
	}
	return false
}

// GetKeyPrefixes returns all unique key prefixes (for tree view).
// Builds the prefix set incrementally to avoid holding all keys in memory.
func (c *Client) GetKeyPrefixes(separator string, maxDepth int) ([]string, error) {
	prefixes := make(map[string]bool)

	err := c.scanEach("*", scanBatchPrefixes, func(keys []string) bool {
		for _, key := range keys {
			parts := strings.Split(key, separator)
			for i := 1; i <= len(parts) && i <= maxDepth; i++ {
				prefix := strings.Join(parts[:i], separator)
				prefixes[prefix] = true
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(prefixes))
	for p := range prefixes {
		result = append(result, p)
	}
	sort.Strings(result)

	return result, nil
}

// detectStringSubtypes checks string-typed keys for HLL/protobuf/bitmap subtypes
// using a pipeline GETRANGE of the first subtypeProbeBytes bytes. Incomplete
// binary probes that may be s2-compressed protobuf are re-fetched in full when
// the key is under subtypeFullFetchMax so the list type is correct without
// selecting the key.
func (c *Client) detectStringSubtypes(keys []types.RedisKey) []types.RedisKey {
	// Collect indices of string keys
	var stringIdxs []int
	for i := range keys {
		if keys[i].Type == "string" {
			stringIdxs = append(stringIdxs, i)
		}
	}
	if len(stringIdxs) == 0 {
		return keys
	}

	pipe := c.pipeline()
	getCmds := make([]*redis.StringCmd, len(stringIdxs))
	for j, idx := range stringIdxs {
		getCmds[j] = pipe.GetRange(c.ctx, keys[idx].Key, 0, subtypeProbeBytes-1)
	}
	_, _ = pipe.Exec(c.ctx)

	// Ambiguous: probe filled the window and looked binary but was not yet
	// classifiable (common for s2+protobuf — needs the whole blob).
	var ambiguous []int
	for j, idx := range stringIdxs {
		val, err := getCmds[j].Result()
		if err != nil {
			continue
		}
		if strings.HasPrefix(val, "HYLL") {
			keys[idx].Type = types.KeyTypeHyperLogLog
			continue
		}
		if decode.LooksLikeProtobuf([]byte(val)) {
			keys[idx].Type = types.KeyTypeProtobuf
			continue
		}
		// A full-size probe may split a multi-byte UTF-8 rune at the cut point.
		binary := isBinaryString(val)
		if len(val) == subtypeProbeBytes {
			binary = isBinaryPrefix(val)
		}
		if !binary {
			continue
		}
		if len(val) < subtypeProbeBytes {
			// Complete short binary value → bitmap.
			keys[idx].Type = types.KeyTypeBitmap
			continue
		}
		ambiguous = append(ambiguous, idx)
	}

	if len(ambiguous) == 0 {
		return keys
	}

	// Second pass: STRLEN, then full GET only for small-enough binary keys.
	pipe = c.pipeline()
	lenCmds := make([]*redis.IntCmd, len(ambiguous))
	for j, idx := range ambiguous {
		lenCmds[j] = pipe.StrLen(c.ctx, keys[idx].Key)
	}
	_, _ = pipe.Exec(c.ctx)

	var fetchIdxs []int
	for j, idx := range ambiguous {
		n, err := lenCmds[j].Result()
		if err != nil || n <= 0 || n > subtypeFullFetchMax {
			continue
		}
		fetchIdxs = append(fetchIdxs, idx)
	}
	if len(fetchIdxs) == 0 {
		return keys
	}

	pipe = c.pipeline()
	fullCmds := make([]*redis.StringCmd, len(fetchIdxs))
	for j, idx := range fetchIdxs {
		fullCmds[j] = pipe.Get(c.ctx, keys[idx].Key)
	}
	_, _ = pipe.Exec(c.ctx)

	for j, idx := range fetchIdxs {
		val, err := fullCmds[j].Result()
		if err != nil {
			continue
		}
		if decode.LooksLikeProtobuf([]byte(val)) {
			keys[idx].Type = types.KeyTypeProtobuf
			continue
		}
		if isBinaryString(val) {
			keys[idx].Type = types.KeyTypeBitmap
		}
	}

	return keys
}

// detectZSetSubtypes checks zset-typed keys for Geo subtype by pipelining
// ZRangeWithScores and checking if scores look like geohash integers.
func (c *Client) detectZSetSubtypes(keys []types.RedisKey) []types.RedisKey {
	var zsetIdxs []int
	for i := range keys {
		if keys[i].Type == "zset" {
			zsetIdxs = append(zsetIdxs, i)
		}
	}
	if len(zsetIdxs) == 0 {
		return keys
	}

	pipe := c.pipeline()
	zrangeCmds := make([]*redis.ZSliceCmd, len(zsetIdxs))
	for j, idx := range zsetIdxs {
		zrangeCmds[j] = pipe.ZRangeWithScores(c.ctx, keys[idx].Key, 0, 0) // only first member
	}
	_, _ = pipe.Exec(c.ctx)

	for j, idx := range zsetIdxs {
		vals, err := zrangeCmds[j].Result()
		if err != nil || len(vals) == 0 {
			continue
		}
		member := types.ZSetMember{
			Member: vals[0].Member.(string),
			Score:  vals[0].Score,
		}
		if looksLikeGeoScores([]types.ZSetMember{member}) {
			keys[idx].Type = types.KeyTypeGeo
		}
	}

	return keys
}
