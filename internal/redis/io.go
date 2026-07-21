package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/davidbudnick/redis-tui/internal/decode"
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// ExportKeys exports keys matching a pattern to a map
func (c *Client) ExportKeys(pattern string) (map[string]any, error) {
	allKeys, err := c.scanAll(pattern, 100)
	if err != nil {
		return nil, err
	}

	export := make(map[string]any)
	chunkSize := 100

	for i := 0; i < len(allKeys); i += chunkSize {
		end := min(i+chunkSize, len(allKeys))
		chunk := allKeys[i:end]

		// Pipeline TYPE + TTL for each key in chunk
		pipe := c.pipeline()
		typeCmds := make([]*redis.StatusCmd, len(chunk))
		ttlCmds := make([]*redis.DurationCmd, len(chunk))
		for j, key := range chunk {
			typeCmds[j] = pipe.Type(c.ctx, key)
			ttlCmds[j] = pipe.TTL(c.ctx, key)
		}
		_, _ = pipe.Exec(c.ctx)

		// Group keys by type for value fetching
		type keyMeta struct {
			key     string
			keyType string
			ttl     time.Duration
		}
		metas := make([]keyMeta, 0, len(chunk))
		for j, key := range chunk {
			kt := typeCmds[j].Val()
			ttl := max(ttlCmds[j].Val(), 0)
			metas = append(metas, keyMeta{key: key, keyType: kt, ttl: ttl})
		}

		// Pipeline value fetches grouped by type
		pipe = c.pipeline()
		type valueFetch struct {
			meta keyMeta
			cmd  any
		}
		fetches := make([]valueFetch, 0, len(metas))
		for _, m := range metas {
			var cmd any
			switch m.keyType {
			case "string":
				cmd = pipe.Get(c.ctx, m.key)
			case "list":
				cmd = pipe.LRange(c.ctx, m.key, 0, -1)
			case "set":
				cmd = pipe.SMembers(c.ctx, m.key)
			case "zset":
				cmd = pipe.ZRangeWithScores(c.ctx, m.key, 0, -1)
			case "hash":
				cmd = pipe.HGetAll(c.ctx, m.key)
			case "stream":
				cmd = pipe.XRange(c.ctx, m.key, "-", "+")
			case "ReJSON-RL":
				cmd = pipe.Do(c.ctx, "JSON.GET", m.key, "$")
			default:
				continue
			}
			fetches = append(fetches, valueFetch{meta: m, cmd: cmd})
		}
		_, _ = pipe.Exec(c.ctx)

		// Collect results
		for _, f := range fetches {
			keyData := map[string]any{
				"type": f.meta.keyType,
				"ttl":  f.meta.ttl.Seconds(),
			}

			switch f.meta.keyType {
			case "string":
				if cmd, ok := f.cmd.(*redis.StringCmd); ok && cmd.Err() == nil {
					keyData["value"] = cmd.Val()
				} else {
					continue
				}
			case "list":
				if cmd, ok := f.cmd.(*redis.StringSliceCmd); ok && cmd.Err() == nil {
					keyData["value"] = cmd.Val()
				} else {
					continue
				}
			case "set":
				if cmd, ok := f.cmd.(*redis.StringSliceCmd); ok && cmd.Err() == nil {
					keyData["value"] = cmd.Val()
				} else {
					continue
				}
			case "zset":
				if cmd, ok := f.cmd.(*redis.ZSliceCmd); ok && cmd.Err() == nil {
					members := make([]map[string]any, len(cmd.Val()))
					for k, z := range cmd.Val() {
						members[k] = map[string]any{"member": z.Member, "score": z.Score}
					}
					keyData["value"] = members
				} else {
					continue
				}
			case "hash":
				if cmd, ok := f.cmd.(*redis.MapStringStringCmd); ok && cmd.Err() == nil {
					keyData["value"] = cmd.Val()
				} else {
					continue
				}
			case "stream":
				if cmd, ok := f.cmd.(*redis.XMessageSliceCmd); ok && cmd.Err() == nil {
					entries := make([]map[string]any, len(cmd.Val()))
					for k, e := range cmd.Val() {
						entries[k] = map[string]any{"id": e.ID, "fields": e.Values}
					}
					keyData["value"] = entries
				} else {
					continue
				}
			case "ReJSON-RL":
				if cmd, ok := f.cmd.(*redis.Cmd); ok && cmd.Err() == nil {
					val, err := cmd.Text()
					if err == nil {
						keyData["value"] = val
					} else {
						continue
					}
				} else {
					continue
				}
			}

			export[f.meta.key] = keyData
		}
	}

	return export, nil
}

// ImportKeys imports keys from a map
func (c *Client) ImportKeys(data map[string]any) (int, error) {
	count := 0

	for key, keyDataRaw := range data {
		keyData, ok := keyDataRaw.(map[string]any)
		if !ok {
			continue
		}

		keyType, _ := keyData["type"].(string)
		ttlSecs, _ := keyData["ttl"].(float64)
		ttl := time.Duration(ttlSecs) * time.Second

		switch keyType {
		case "string":
			if val, ok := keyData["value"].(string); ok {
				_ = c.SetString(key, val, ttl)
				count++
			}
		case "list":
			if vals, ok := keyData["value"].([]any); ok {
				strs := make([]string, 0, len(vals))
				for _, v := range vals {
					if s, ok := v.(string); ok {
						strs = append(strs, s)
					}
				}
				if len(strs) > 0 {
					_ = c.RPush(key, strs...)
				}
				if ttl > 0 {
					_ = c.SetTTL(key, ttl)
				}
				count++
			}
		case "set":
			if vals, ok := keyData["value"].([]any); ok {
				strs := make([]string, 0, len(vals))
				for _, v := range vals {
					if s, ok := v.(string); ok {
						strs = append(strs, s)
					}
				}
				if len(strs) > 0 {
					_ = c.SAdd(key, strs...)
				}
				if ttl > 0 {
					_ = c.SetTTL(key, ttl)
				}
				count++
			}
		case "zset":
			if vals, ok := keyData["value"].([]any); ok {
				members := make([]redis.Z, 0, len(vals))
				for _, v := range vals {
					if m, ok := v.(map[string]any); ok {
						member, _ := m["member"].(string)
						score, _ := m["score"].(float64)
						members = append(members, redis.Z{Score: score, Member: member})
					}
				}
				if len(members) > 0 {
					_ = c.ZAddBatch(key, members...)
				}
				if ttl > 0 {
					_ = c.SetTTL(key, ttl)
				}
				count++
			}
		case "hash":
			if vals, ok := keyData["value"].(map[string]any); ok {
				fields := make(map[string]string, len(vals))
				for field, val := range vals {
					if s, ok := val.(string); ok {
						fields[field] = s
					}
				}
				if len(fields) > 0 {
					_ = c.HSetMap(key, fields)
				}
				if ttl > 0 {
					_ = c.SetTTL(key, ttl)
				}
				count++
			}
		case "ReJSON-RL":
			if val, ok := keyData["value"].(string); ok {
				_ = c.JSONSet(key, val)
				if ttl > 0 {
					_ = c.SetTTL(key, ttl)
				}
				count++
			}
		}
	}

	return count, nil
}

// CompareKeys compares two keys and returns their values.
// Pipelines both TYPE commands and both value fetches to reduce round-trips from 4 to 2.
func (c *Client) CompareKeys(key1, key2 string) (types.RedisValue, types.RedisValue, error) {
	// Pipeline 1: get both types
	pipe := c.pipeline()
	type1Cmd := pipe.Type(c.ctx, key1)
	type2Cmd := pipe.Type(c.ctx, key2)
	_, err := pipe.Exec(c.ctx)
	if err != nil && err != redis.Nil {
		return types.RedisValue{}, types.RedisValue{}, fmt.Errorf("error getting types: %w", err)
	}

	keyType1, _ := type1Cmd.Result()
	keyType2, _ := type2Cmd.Result()

	// Pipeline 2: get both values based on types
	pipe = c.pipeline()
	cmds1 := queueValueFetch(pipe, c.ctx, key1, keyType1)
	cmds2 := queueValueFetch(pipe, c.ctx, key2, keyType2)
	_, _ = pipe.Exec(c.ctx)

	val1 := extractValue(keyType1, cmds1)
	val2 := extractValue(keyType2, cmds2)

	return val1, val2, nil
}

type valueFetchCmds struct {
	strCmd    *redis.StringCmd
	listCmd   *redis.StringSliceCmd
	setCmd    *redis.StringSliceCmd
	zsetCmd   *redis.ZSliceCmd
	hashCmd   *redis.MapStringStringCmd
	streamCmd *redis.XMessageSliceCmd
	jsonCmd   *redis.Cmd
}

func queueValueFetch(pipe redis.Pipeliner, ctx context.Context, key, keyType string) valueFetchCmds {
	var r valueFetchCmds
	switch keyType {
	case "string":
		r.strCmd = pipe.Get(ctx, key)
	case "list":
		r.listCmd = pipe.LRange(ctx, key, 0, -1)
	case "set":
		r.setCmd = pipe.SMembers(ctx, key)
	case "zset":
		r.zsetCmd = pipe.ZRangeWithScores(ctx, key, 0, -1)
	case "hash":
		r.hashCmd = pipe.HGetAll(ctx, key)
	case "stream":
		r.streamCmd = pipe.XRange(ctx, key, "-", "+")
	case "ReJSON-RL":
		r.jsonCmd = pipe.Do(ctx, "JSON.GET", key, "$")
	}
	return r
}

func extractValue(keyType string, r valueFetchCmds) types.RedisValue {
	var value types.RedisValue
	value.Type = types.KeyType(keyType)

	switch keyType {
	case "string":
		if r.strCmd != nil {
			value.StringValue, _ = r.strCmd.Result()
			if len(value.StringValue) >= 4 && value.StringValue[:4] == "HYLL" {
				value.Type = types.KeyTypeHyperLogLog
			} else if decoded, ok := decode.TryBinary([]byte(value.StringValue)); ok {
				value.Type = types.KeyTypeProtobuf
				value.DecodedValue = decoded.Text
				value.DecodedFormat = decoded.Format
				value.RawSize = decoded.RawSize
				value.DecodedSize = decoded.DecodedSize
			} else if isBinaryString(value.StringValue) {
				value.Type = types.KeyTypeBitmap
				value.BitPositions = extractBitPositions([]byte(value.StringValue), maxBitPositions)
			}
		}
	case "list":
		if r.listCmd != nil {
			value.ListValue, _ = r.listCmd.Result()
		}
	case "set":
		if r.setCmd != nil {
			value.SetValue, _ = r.setCmd.Result()
		}
	case "zset":
		if r.zsetCmd != nil {
			vals, _ := r.zsetCmd.Result()
			for _, z := range vals {
				value.ZSetValue = append(value.ZSetValue, types.ZSetMember{
					Member: z.Member.(string),
					Score:  z.Score,
				})
			}
		}
	case "hash":
		if r.hashCmd != nil {
			value.HashValue, _ = r.hashCmd.Result()
		}
	case "stream":
		if r.streamCmd != nil {
			entries, _ := r.streamCmd.Result()
			for _, entry := range entries {
				value.StreamValue = append(value.StreamValue, types.StreamEntry{
					ID:     entry.ID,
					Fields: entry.Values,
				})
			}
		}
	case "ReJSON-RL":
		if r.jsonCmd != nil {
			value.JSONValue, _ = r.jsonCmd.Text()
		}
	}

	return value
}
