package redis

import (
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
// Scans incrementally to avoid holding the full keyspace in memory.
func (c *Client) FuzzySearchKeys(searchTerm string, maxKeys int) ([]types.RedisKey, error) {
	searchLower := strings.ToLower(searchTerm)

	type scoredKey struct {
		key   string
		score int
	}
	var scoredKeys []scoredKey

	err := c.scanEach("*", scanBatchDefault, func(keys []string) bool {
		for _, key := range keys {
			keyLower := strings.ToLower(key)
			score := fuzzyScore(keyLower, searchLower)
			if score > 0 {
				scoredKeys = append(scoredKeys, scoredKey{key: key, score: score})
			}
		}
		return true // must scan all keys for global top-N
	})
	if err != nil {
		return nil, err
	}

	// Sort by score descending
	sort.Slice(scoredKeys, func(i, j int) bool {
		return scoredKeys[i].score > scoredKeys[j].score
	})

	// Limit to maxKeys
	limit := min(maxKeys, len(scoredKeys))
	scoredKeys = scoredKeys[:limit]

	if len(scoredKeys) == 0 {
		return []types.RedisKey{}, nil
	}

	// Use pipeline to batch Type and TTL calls for top results only
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
// Uses 2 pipelines per chunk (TYPE + values) and defers TTL to a single final pipeline.
func (c *Client) SearchByValue(pattern string, valueSearch string, maxKeys int) ([]types.RedisKey, error) {
	allKeys, err := c.scanAll(pattern, scanBatchDefault)
	if err != nil {
		return nil, err
	}

	type match struct {
		key     string
		keyType string
	}
	matches := make([]match, 0, maxKeys)

	// Process in chunks to keep pipeline sizes reasonable
	chunkSize := int(scanBatchDefault)
	for i := 0; i < len(allKeys) && len(matches) < maxKeys; i += chunkSize {
		end := min(i+chunkSize, len(allKeys))
		keys := allKeys[i:end]

		// Pipeline 1: get types for all keys
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

		// Pipeline 2: get values based on type
		valuePipe := c.pipeline()
		type valueCmd struct {
			idx     int
			keyType string
			strCmd  *redis.StringCmd
			hashCmd *redis.MapStringStringCmd
			listCmd *redis.StringSliceCmd
			setCmd  *redis.StringSliceCmd
			jsonCmd *redis.Cmd
		}
		valueCmds := make([]valueCmd, 0, len(keys))

		for j, key := range keys {
			kt := keyTypes[j]
			vc := valueCmd{idx: j, keyType: kt}
			switch kt {
			case "string":
				vc.strCmd = valuePipe.Get(c.ctx, key)
			case "hash":
				vc.hashCmd = valuePipe.HGetAll(c.ctx, key)
			case "list":
				vc.listCmd = valuePipe.LRange(c.ctx, key, 0, -1)
			case "set":
				vc.setCmd = valuePipe.SMembers(c.ctx, key)
			case "ReJSON-RL":
				vc.jsonCmd = valuePipe.Do(c.ctx, "JSON.GET", key, "$")
			default:
				continue
			}
			valueCmds = append(valueCmds, vc)
		}
		_, _ = valuePipe.Exec(c.ctx)

		// Find matching keys
		for _, vc := range valueCmds {
			found := false
			switch vc.keyType {
			case "string":
				val, _ := vc.strCmd.Result()
				found = strings.Contains(val, valueSearch)
			case "hash":
				vals, _ := vc.hashCmd.Result()
				for _, v := range vals {
					if strings.Contains(v, valueSearch) {
						found = true
						break
					}
				}
			case "list":
				vals, _ := vc.listCmd.Result()
				for _, v := range vals {
					if strings.Contains(v, valueSearch) {
						found = true
						break
					}
				}
			case "set":
				vals, _ := vc.setCmd.Result()
				for _, v := range vals {
					if strings.Contains(v, valueSearch) {
						found = true
						break
					}
				}
			case "ReJSON-RL":
				val, _ := vc.jsonCmd.Text()
				found = strings.Contains(val, valueSearch)
			}
			if found {
				matches = append(matches, match{key: keys[vc.idx], keyType: keyTypes[vc.idx]})
				if len(matches) >= maxKeys {
					break
				}
			}
		}
	}

	if len(matches) == 0 {
		return []types.RedisKey{}, nil
	}

	// Single final pipeline for TTL of all matches
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
// using a pipeline GETRANGE of the first subtypeProbeBytes bytes — a full GET
// would transfer entire (possibly huge) values on every scan page.
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
		// Incomplete probes of large binary values may be compressed protobuf
		// that only decodes with the full payload — leave type as string.
		if binary && len(val) < subtypeProbeBytes {
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
