package redis

import (
	"strings"

	"github.com/davidbudnick/redis-tui/internal/decode"
	"github.com/davidbudnick/redis-tui/internal/types"
)

const (
	previewMaxItems       = 100
	previewMaxStringBytes = 64 * 1024
	detailMaxItems        = 1000
	detailMaxStringBytes  = 1024 * 1024
)

// GetValuePreview retrieves a bounded preview of a key's value. Unlike
// GetValue it never fetches entire collections (LRANGE 0 -1, HGETALL,
// SMEMBERS, ...), so selecting a huge key in the keys list cannot block the
// Redis server or transfer megabytes for a screenful of preview. When the
// value was cut off, Truncated is set and TotalCount holds the full
// length/cardinality.
func (c *Client) GetValuePreview(key string) (types.RedisValue, error) {
	return c.getValueBounded(key, previewMaxItems, previewMaxStringBytes, true)
}

// markTruncated sets the Truncated/TotalCount fields when fewer entries were
// fetched than exist.
func markTruncated(value *types.RedisValue, total int64, fetched int) {
	if total > int64(fetched) {
		value.Truncated = true
		value.TotalCount = total
	}
}

// getValueBounded loads a key with collection/string caps. When deferTruncatedBinary
// is true, truncated non-protobuf binary stays typed as string (preview path).
func (c *Client) getValueBounded(key string, maxItems int, maxStringBytes int64, deferTruncatedBinary bool) (types.RedisValue, error) {
	keyType, err := c.cmdable().Type(c.ctx, key).Result()
	if err != nil {
		return types.RedisValue{}, err
	}

	var value types.RedisValue
	value.Type = types.KeyType(keyType)

	switch keyType {
	case "string":
		if err := c.fetchString(key, &value, maxStringBytes, deferTruncatedBinary); err != nil {
			return value, err
		}

	case "list":
		if err := c.fetchList(key, &value, maxItems); err != nil {
			return value, err
		}

	case "set":
		if err := c.fetchSet(key, &value, maxItems); err != nil {
			return value, err
		}

	case "zset":
		if err := c.fetchZSet(key, &value, maxItems); err != nil {
			return value, err
		}

	case "hash":
		if err := c.fetchHash(key, &value, maxItems); err != nil {
			return value, err
		}

	case "stream":
		if err := c.fetchStream(key, &value, maxItems); err != nil {
			return value, err
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

// fetchString fills a bounded string value, including HLL/protobuf/bitmap detection.
func (c *Client) fetchString(key string, value *types.RedisValue, maxBytes int64, deferTruncatedBinary bool) error {
	total, err := c.cmdable().StrLen(c.ctx, key).Result()
	if err != nil {
		return err
	}
	val, err := c.cmdable().GetRange(c.ctx, key, 0, maxBytes-1).Result()
	if err != nil {
		return err
	}
	value.StringValue = val
	markTruncated(value, total, len(val))

	if strings.HasPrefix(val, "HYLL") {
		value.Type = types.KeyTypeHyperLogLog
		if count, err := c.cmdable().PFCount(c.ctx, key).Result(); err == nil {
			value.HLLCount = count
		}
		return nil
	}

	if decoded, ok := decode.TryBinary([]byte(val)); ok {
		value.Type = types.KeyTypeProtobuf
		value.DecodedValue = decoded.Text
		value.DecodedFormat = decoded.Format
		value.RawSize = decoded.RawSize
		value.DecodedSize = decoded.DecodedSize
		return nil
	}

	binary := isBinaryString(val)
	if value.Truncated {
		binary = isBinaryPrefix(val)
		if binary && deferTruncatedBinary {
			return nil
		}
	}
	if binary {
		value.Type = types.KeyTypeBitmap
		if count, err := c.cmdable().BitCount(c.ctx, key, nil).Result(); err == nil {
			value.BitCount = count
		}
		value.BitPositions = extractBitPositions([]byte(val), maxBitPositions)
	}
	return nil
}

// fetchList loads up to maxItems list elements.
func (c *Client) fetchList(key string, value *types.RedisValue, maxItems int) error {
	total, err := c.cmdable().LLen(c.ctx, key).Result()
	if err != nil {
		return err
	}
	vals, err := c.cmdable().LRange(c.ctx, key, 0, int64(maxItems)-1).Result()
	if err != nil {
		return err
	}
	value.ListValue = vals
	markTruncated(value, total, len(vals))
	return nil
}

// fetchSet loads up to maxItems set members via SSCAN.
func (c *Client) fetchSet(key string, value *types.RedisValue, maxItems int) error {
	total, err := c.cmdable().SCard(c.ctx, key).Result()
	if err != nil {
		return err
	}
	members, err := c.scanSetMembers(key, maxItems)
	if err != nil {
		return err
	}
	value.SetValue = members
	markTruncated(value, total, len(members))
	return nil
}

// fetchZSet loads up to maxItems sorted-set members and detects geo keys.
func (c *Client) fetchZSet(key string, value *types.RedisValue, maxItems int) error {
	total, err := c.cmdable().ZCard(c.ctx, key).Result()
	if err != nil {
		return err
	}
	vals, err := c.cmdable().ZRangeWithScores(c.ctx, key, 0, int64(maxItems)-1).Result()
	if err != nil {
		return err
	}
	for _, z := range vals {
		value.ZSetValue = append(value.ZSetValue, types.ZSetMember{
			Member: z.Member.(string),
			Score:  z.Score,
		})
	}
	markTruncated(value, total, len(value.ZSetValue))
	c.detectGeo(key, value)
	return nil
}

// fetchHash loads up to maxItems hash fields via HSCAN.
func (c *Client) fetchHash(key string, value *types.RedisValue, maxItems int) error {
	total, err := c.cmdable().HLen(c.ctx, key).Result()
	if err != nil {
		return err
	}
	fields, err := c.scanHashFields(key, maxItems)
	if err != nil {
		return err
	}
	value.HashValue = fields
	markTruncated(value, total, len(fields))
	return nil
}

// fetchStream loads up to maxItems stream entries.
func (c *Client) fetchStream(key string, value *types.RedisValue, maxItems int) error {
	total, err := c.cmdable().XLen(c.ctx, key).Result()
	if err != nil {
		return err
	}
	entries, err := c.cmdable().XRangeN(c.ctx, key, "-", "+", int64(maxItems)).Result()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		value.StreamValue = append(value.StreamValue, types.StreamEntry{
			ID:     entry.ID,
			Fields: entry.Values,
		})
	}
	markTruncated(value, total, len(value.StreamValue))
	return nil
}

// scanSetMembers collects up to maxItems set members via SSCAN.
func (c *Client) scanSetMembers(key string, maxItems int) ([]string, error) {
	members := make([]string, 0, maxItems)
	var cursor uint64
	for len(members) < maxItems {
		batch, next, err := c.cmdable().SScan(c.ctx, key, cursor, "", int64(maxItems)).Result()
		if err != nil {
			return nil, err
		}
		for _, m := range batch {
			members = append(members, m)
			if len(members) >= maxItems {
				break
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return members, nil
}

// scanHashFields collects up to maxItems hash fields via HSCAN.
func (c *Client) scanHashFields(key string, maxItems int) (map[string]string, error) {
	fields := make(map[string]string, maxItems)
	var cursor uint64
	for len(fields) < maxItems {
		batch, next, err := c.cmdable().HScan(c.ctx, key, cursor, "", int64(maxItems)).Result()
		if err != nil {
			return nil, err
		}
		for i := 0; i+1 < len(batch) && len(fields) < maxItems; i += 2 {
			fields[batch[i]] = batch[i+1]
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return fields, nil
}

// detectGeo flips a zset to Geo when scores look like geohash integers.
func (c *Client) detectGeo(key string, value *types.RedisValue) {
	if len(value.ZSetValue) == 0 || !looksLikeGeoScores(value.ZSetValue) {
		return
	}
	members := make([]string, len(value.ZSetValue))
	for i, m := range value.ZSetValue {
		members[i] = m.Member
	}
	positions, err := c.cmdable().GeoPos(c.ctx, key, members...).Result()
	if err != nil {
		return
	}
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
