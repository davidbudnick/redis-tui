package redis

import (
	"strings"

	"github.com/davidbudnick/redis-tui/internal/decode"
	"github.com/davidbudnick/redis-tui/internal/types"
)

const (
	// previewMaxItems caps how many collection entries are fetched for the
	// keys-list preview panel, which only ever displays a screenful.
	previewMaxItems = 100
	// previewMaxStringBytes caps how many bytes of a string value are fetched
	// for the preview panel.
	previewMaxStringBytes = 64 * 1024
)

// GetValuePreview retrieves a bounded preview of a key's value. Unlike
// GetValue it never fetches entire collections (LRANGE 0 -1, HGETALL,
// SMEMBERS, ...), so selecting a huge key in the keys list cannot block the
// Redis server or transfer megabytes for a screenful of preview. When the
// value was cut off, Truncated is set and TotalCount holds the full
// length/cardinality.
func (c *Client) GetValuePreview(key string) (types.RedisValue, error) {
	keyType, err := c.cmdable().Type(c.ctx, key).Result()
	if err != nil {
		return types.RedisValue{}, err
	}

	var value types.RedisValue
	value.Type = types.KeyType(keyType)

	switch keyType {
	case "string":
		if err := c.previewString(key, &value); err != nil {
			return value, err
		}

	case "list":
		total, err := c.cmdable().LLen(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		vals, err := c.cmdable().LRange(c.ctx, key, 0, previewMaxItems-1).Result()
		if err != nil {
			return value, err
		}
		value.ListValue = vals
		markTruncated(&value, total, len(vals))

	case "set":
		total, err := c.cmdable().SCard(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		members, err := c.scanSetPreview(key)
		if err != nil {
			return value, err
		}
		value.SetValue = members
		markTruncated(&value, total, len(members))

	case "zset":
		total, err := c.cmdable().ZCard(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		vals, err := c.cmdable().ZRangeWithScores(c.ctx, key, 0, previewMaxItems-1).Result()
		if err != nil {
			return value, err
		}
		for _, z := range vals {
			value.ZSetValue = append(value.ZSetValue, types.ZSetMember{
				Member: z.Member.(string),
				Score:  z.Score,
			})
		}
		markTruncated(&value, total, len(value.ZSetValue))
		c.previewGeoDetect(key, &value)

	case "hash":
		total, err := c.cmdable().HLen(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		fields, err := c.scanHashPreview(key)
		if err != nil {
			return value, err
		}
		value.HashValue = fields
		markTruncated(&value, total, len(fields))

	case "stream":
		total, err := c.cmdable().XLen(c.ctx, key).Result()
		if err != nil {
			return value, err
		}
		entries, err := c.cmdable().XRangeN(c.ctx, key, "-", "+", previewMaxItems).Result()
		if err != nil {
			return value, err
		}
		for _, entry := range entries {
			value.StreamValue = append(value.StreamValue, types.StreamEntry{
				ID:     entry.ID,
				Fields: entry.Values,
			})
		}
		markTruncated(&value, total, len(value.StreamValue))

	case "ReJSON-RL":
		val, err := c.do("JSON.GET", key, "$").Text()
		if err != nil {
			return value, err
		}
		value.JSONValue = val
	}

	return value, nil
}

// markTruncated sets the Truncated/TotalCount fields when fewer entries were
// fetched than exist.
func markTruncated(value *types.RedisValue, total int64, fetched int) {
	if total > int64(fetched) {
		value.Truncated = true
		value.TotalCount = total
	}
}

// previewString fills in a bounded string preview, including HLL, protobuf, and
// bitmap subtype detection on the fetched bytes.
func (c *Client) previewString(key string, value *types.RedisValue) error {
	total, err := c.cmdable().StrLen(c.ctx, key).Result()
	if err != nil {
		return err
	}
	val, err := c.cmdable().GetRange(c.ctx, key, 0, previewMaxStringBytes-1).Result()
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

	// Prefer protobuf (incl. s2-compressed) over bitmap: binary probes that are
	// really compressed menus must not be labeled as bitmaps in the list UI.
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
		// Incomplete binary blob may still be compressed protobuf that only
		// decodes with the full payload — leave type as string until GetValue.
		if binary {
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

// scanSetPreview collects roughly previewMaxItems set members from a single
// SSCAN iteration. COUNT is a hint, so slightly more or fewer entries may be
// returned — the preview panel only displays a screenful either way.
func (c *Client) scanSetPreview(key string) ([]string, error) {
	members, _, err := c.cmdable().SScan(c.ctx, key, 0, "", previewMaxItems).Result()
	return members, err
}

// scanHashPreview collects roughly previewMaxItems hash fields from a single
// HSCAN iteration (returned as flattened field/value pairs). COUNT is only a
// hint, so servers may return more pairs than asked; inserts are capped so the
// preview never materializes an unbounded map.
func (c *Client) scanHashPreview(key string) (map[string]string, error) {
	batch, _, err := c.cmdable().HScan(c.ctx, key, 0, "", previewMaxItems).Result()
	if err != nil {
		return nil, err
	}
	fields := make(map[string]string, previewMaxItems)
	for i := 0; i+1 < len(batch) && len(fields) < previewMaxItems; i += 2 {
		fields[batch[i]] = batch[i+1]
	}
	return fields, nil
}

// previewGeoDetect flips a zset preview to Geo when its scores look like
// geohash integers, resolving coordinates for the previewed members.
func (c *Client) previewGeoDetect(key string, value *types.RedisValue) {
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
