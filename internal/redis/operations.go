package redis

import (
	"time"
	"unicode/utf8"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// maxBitPositions caps how many set-bit positions are extracted from a bitmap
// value — the UI displays only a screenful, and an unbounded extraction of a
// large bitmap could allocate millions of entries.
const maxBitPositions = 1024

// GetValue retrieves a bounded detail value for a key (never unbounded dumps).
func (c *Client) GetValue(key string) (types.RedisValue, error) {
	return c.getValueBounded(key, detailMaxItems, detailMaxStringBytes, false)
}

// looksLikeGeoScores returns true if all sorted set scores look like 52-bit
// geohash integers produced by GEOADD (range ~1e14 to ~5e15, integer values).
// Regular ZADD scores (e.g. 1.5, 100, 9850) are much smaller.
func looksLikeGeoScores(members []types.ZSetMember) bool {
	for _, m := range members {
		s := m.Score
		if s < 1e14 || s > 5e15 {
			return false
		}
		if s != float64(int64(s)) {
			return false
		}
	}
	return true
}

// isBinaryString returns true if the string contains binary data (invalid
// UTF-8 or null bytes), suggesting it was created via SETBIT as a bitmap.
func isBinaryString(s string) bool {
	if len(s) == 0 {
		return false
	}
	return !utf8.ValidString(s)
}

// extractBitPositions returns up to max set bit offsets from raw bytes.
func extractBitPositions(val []byte, max int) []int64 {
	if max <= 0 {
		return nil
	}
	var positions []int64
	for byteIdx := 0; byteIdx < len(val); byteIdx++ {
		b := val[byteIdx]
		for bit := 7; bit >= 0; bit-- {
			if b&(1<<uint(bit)) != 0 {
				positions = append(positions, int64(byteIdx*8+(7-bit)))
				if len(positions) >= max {
					return positions
				}
			}
		}
	}
	return positions
}

// isBinaryPrefix is like isBinaryString but for a value prefix (e.g. from
// GETRANGE): a multi-byte UTF-8 rune split at the cut point must not count as
// binary, so up to utf8.UTFMax-1 trailing bytes of an incomplete rune are
// trimmed before validating.
func isBinaryPrefix(s string) bool {
	for i := 0; i < utf8.UTFMax-1 && len(s) > 0; i++ {
		if utf8.ValidString(s) {
			return false
		}
		r, size := utf8.DecodeLastRuneInString(s)
		if r != utf8.RuneError || size != 1 {
			break
		}
		s = s[:len(s)-1]
	}
	return isBinaryString(s)
}

// JSONGet retrieves a JSON value from a RedisJSON key
func (c *Client) JSONGet(key string) (string, error) {
	return c.do("JSON.GET", key, "$").Text()
}

// JSONGetPath retrieves a JSON value at a specific path from a RedisJSON key
func (c *Client) JSONGetPath(key, path string) (string, error) {
	return c.do("JSON.GET", key, path).Text()
}

// JSONSet sets a JSON value on a RedisJSON key
func (c *Client) JSONSet(key, value string) error {
	return c.do("JSON.SET", key, "$", value).Err()
}

// DeleteKey deletes a single key
func (c *Client) DeleteKey(key string) error {
	return c.cmdable().Del(c.ctx, key).Err()
}

// DeleteKeys deletes multiple keys
func (c *Client) DeleteKeys(keys ...string) (int64, error) {
	return c.cmdable().Del(c.ctx, keys...).Result()
}

// BulkDelete deletes all keys matching a pattern
func (c *Client) BulkDelete(pattern string) (int, error) {
	allKeys, err := c.scanAll(pattern, 100)
	if err != nil {
		return 0, err
	}

	var deleted int
	// Delete in chunks to avoid huge DEL commands
	chunkSize := 100
	for i := 0; i < len(allKeys); i += chunkSize {
		end := min(i+chunkSize, len(allKeys))
		count, err := c.cmdable().Del(c.ctx, allKeys[i:end]...).Result()
		if err != nil {
			return deleted, err
		}
		deleted += int(count)
	}

	return deleted, nil
}

// SetString sets a string value
func (c *Client) SetString(key, value string, ttl time.Duration) error {
	return c.cmdable().Set(c.ctx, key, value, ttl).Err()
}

// SetTTL sets or removes TTL on a key
func (c *Client) SetTTL(key string, ttl time.Duration) error {
	if ttl <= 0 {
		return c.cmdable().Persist(c.ctx, key).Err()
	}
	return c.cmdable().Expire(c.ctx, key, ttl).Err()
}

// BatchSetTTL sets TTL on all keys matching a pattern
func (c *Client) BatchSetTTL(pattern string, ttl time.Duration) (int, error) {
	allKeys, err := c.scanAll(pattern, 100)
	if err != nil {
		return 0, err
	}

	var count int
	// Process in chunks to keep pipeline sizes reasonable
	chunkSize := 100
	for i := 0; i < len(allKeys); i += chunkSize {
		end := min(i+chunkSize, len(allKeys))
		keys := allKeys[i:end]

		pipe := c.pipeline()
		cmds := make([]*redis.BoolCmd, len(keys))

		for j, key := range keys {
			if ttl <= 0 {
				cmds[j] = pipe.Persist(c.ctx, key)
			} else {
				cmds[j] = pipe.Expire(c.ctx, key, ttl)
			}
		}

		_, _ = pipe.Exec(c.ctx)

		for _, cmd := range cmds {
			if cmd.Err() == nil {
				count++
			}
		}
	}

	return count, nil
}

// MemoryUsage returns memory usage for a key
func (c *Client) MemoryUsage(key string) (int64, error) {
	return c.cmdable().MemoryUsage(c.ctx, key).Result()
}

// Rename renames a key
func (c *Client) Rename(oldKey, newKey string) error {
	return c.cmdable().Rename(c.ctx, oldKey, newKey).Err()
}

// Copy copies a key within the currently selected database.
func (c *Client) Copy(src, dst string, replace bool) error {
	c.mu.RLock()
	db := c.db
	c.mu.RUnlock()
	return c.cmdable().Copy(c.ctx, src, dst, db, replace).Err()
}
