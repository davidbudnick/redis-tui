package redis

import (
	"time"
	"unicode/utf8"

	"github.com/davidbudnick/redis-tui/internal/decode"
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// maxBitPositions caps how many set-bit offsets we materialize for display.
const maxBitPositions = 256

// GetValue retrieves the value for a key
func (c *Client) GetValue(key string) (types.RedisValue, error) {
	keyType, err := c.cmdable().Type(c.ctx, key).Result()
	if err != nil {
		return types.RedisValue{}, err
	}

	var value types.RedisValue
	value.Type = types.KeyType(keyType)

	switch keyType {
	case "string":
		val, err := c.cmdable().Get(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		value.StringValue = val

		// Detect HyperLogLog: raw value starts with "HYLL" magic bytes
		if len(val) >= 4 && val[:4] == "HYLL" {
			value.Type = types.KeyTypeHyperLogLog
			count, err := c.cmdable().PFCount(c.ctx, key).Result()
			if err == nil {
				value.HLLCount = count
			}
		} else if decoded, ok := decode.TryBinary([]byte(val)); ok {
			value.Type = types.KeyTypeProtobuf
			value.DecodedValue = decoded.Text
			value.DecodedFormat = decoded.Format
			value.RawSize = decoded.RawSize
			value.DecodedSize = decoded.DecodedSize
		} else if isBinaryString(val) {
			applyBinaryStringValue(&value, val, c, key)
		}

	case "list":
		vals, err := c.cmdable().LRange(c.ctx, key, 0, -1).Result()
		if err != nil {
			return value, err
		}
		value.ListValue = vals

	case "set":
		vals, err := c.cmdable().SMembers(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		value.SetValue = vals

	case "zset":
		vals, err := c.cmdable().ZRangeWithScores(c.ctx, key, 0, -1).Result()
		if err != nil {
			return value, err
		}
		for _, z := range vals {
			value.ZSetValue = append(value.ZSetValue, types.ZSetMember{
				Member: z.Member.(string),
				Score:  z.Score,
			})
		}

		// Detect Geo: check if scores look like 52-bit geohash integers
		if len(value.ZSetValue) > 0 && looksLikeGeoScores(value.ZSetValue) {
			members := make([]string, len(value.ZSetValue))
			for i, m := range value.ZSetValue {
				members[i] = m.Member
			}
			positions, err := c.cmdable().GeoPos(c.ctx, key, members...).Result()
			if err == nil {
				var geoMembers []types.GeoMember
				for i, pos := range positions {
					if pos != nil {
						geoMembers = append(geoMembers, types.GeoMember{
							Name:      members[i],
							Longitude: pos.Longitude,
							Latitude:  pos.Latitude,
						})
					}
				}
				if len(geoMembers) > 0 {
					value.Type = types.KeyTypeGeo
					value.GeoValue = geoMembers
				}
			}
		}

	case "hash":
		vals, err := c.cmdable().HGetAll(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		value.HashValue = vals

	case "stream":
		entries, err := c.cmdable().XRange(c.ctx, key, "-", "+").Result()
		if err != nil {
			return value, err
		}
		for _, entry := range entries {
			value.StreamValue = append(value.StreamValue, types.StreamEntry{
				ID:     entry.ID,
				Fields: entry.Values,
			})
		}

	case "ReJSON-RL":
		val, err := c.do("JSON.GET", key, "$").Text()
		if err != nil {
			return value, err
		}
		value.JSONValue = val
	}

	return value, nil
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

// applyBinaryStringValue classifies remaining binary Redis strings as bitmaps.
func applyBinaryStringValue(value *types.RedisValue, val string, c *Client, key string) {
	value.Type = types.KeyTypeBitmap
	count, err := c.cmdable().BitCount(c.ctx, key, &redis.BitCount{Start: 0, End: -1}).Result()
	if err == nil {
		value.BitCount = count
	}
	value.BitPositions = extractBitPositions([]byte(val), maxBitPositions)
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
