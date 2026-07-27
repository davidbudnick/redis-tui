// Command seed populates Redis instances started by the example docker-compose
// files with sample data covering every data type. Run with:
//
//	go run ./examples/seed                          # standalone (localhost:6379)
//	go run ./examples/seed -addr localhost:6380 -cluster  # cluster
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/klauspost/compress/s2"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protowire"
)

func main() {
	addr := flag.String("addr", "localhost:6379", "Redis address (host:port)")
	cluster := flag.Bool("cluster", false, "Connect in cluster mode")
	flush := flag.Bool("flush", false, "Flush all data before seeding")
	flag.Parse()

	run(*addr, *cluster, *flush)
}

func run(addr string, cluster, flush bool) {
	ctx := context.Background()

	var rdb redis.Cmdable
	if cluster {
		rdb = makeClusterClient(ctx, addr)
	} else {
		rdb = redis.NewClient(&redis.Options{Addr: addr})
	}

	if err := rdb.(interface{ Ping(context.Context) *redis.StatusCmd }).Ping(ctx).Err(); err != nil {
		logFatalf("cannot connect to %s: %v", addr, err)
	}
	fmt.Printf("Connected to %s\n", addr)

	if flush {
		flushAll(ctx, rdb)
	}

	runSeeds(ctx, rdb)
}

func flushAll(ctx context.Context, rdb redis.Cmdable) {
	switch c := rdb.(type) {
	case *redis.Client:
		if err := c.FlushAll(ctx).Err(); err != nil {
			logFatalf("flush failed: %v", err)
		}
	case *redis.ClusterClient:
		if err := c.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			return client.FlushAll(ctx).Err()
		}); err != nil {
			logFatalf("flush failed: %v", err)
		}
	}
	fmt.Println("Flushed existing data")
}

func runSeeds(ctx context.Context, rdb redis.Cmdable) {
	seedStrings(ctx, rdb)
	seedLists(ctx, rdb)
	seedSets(ctx, rdb)
	seedSortedSets(ctx, rdb)
	seedHashes(ctx, rdb)
	seedStreams(ctx, rdb)
	seedHyperLogLog(ctx, rdb)
	seedBitmaps(ctx, rdb)
	seedGeo(ctx, rdb)
	seedTTLKeys(ctx, rdb)
	seedNestedKeys(ctx, rdb)
	seedJSONStrings(ctx, rdb)
	seedProtobuf(ctx, rdb)
	if checkJSON(ctx, rdb) {
		seedJSON(ctx, rdb)
	} else {
		fmt.Println("  json (native): skipped — RedisJSON module not available")
	}

	fmt.Println("Done — seeding complete")
}

// newClusterClient creates a cluster client that remaps node addresses from
// Docker-internal IPs to the provided host. This is needed because cluster
// nodes running in Docker advertise their container IPs (172.x.x.x) which
// are unreachable from the host.
func newClusterClient(ctx context.Context, addr string) *redis.ClusterClient {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	// Discover slot-to-port mapping from the seed node, then rewrite
	// the advertised addresses to use the provided host.
	node := redis.NewClient(&redis.Options{Addr: addr})
	defer node.Close()

	slots, err := node.ClusterSlots(ctx).Result()
	if err != nil {
		logFatalf("cannot read cluster slots from %s: %v", addr, err)
	}

	for i := range slots {
		for j := range slots[i].Nodes {
			_, port, _ := net.SplitHostPort(slots[i].Nodes[j].Addr)
			slots[i].Nodes[j].Addr = net.JoinHostPort(host, port)
		}
	}

	return redis.NewClusterClient(&redis.ClusterOptions{
		ClusterSlots: func(context.Context) ([]redis.ClusterSlot, error) {
			return slots, nil
		},
	})
}

func seedStrings(ctx context.Context, rdb redis.Cmdable) {
	data := map[string]string{
		"app:name":            "redis-tui",
		"app:version":         "1.0.17",
		"app:env":             "development",
		"feature:dark-mode":   "enabled",
		"feature:beta-search": "disabled",
		"counter:visits":      "48291",
		"counter:errors":      "17",
		"counter:signups":     "3042",
		"msg:welcome":         "Welcome to Redis TUI! Browse, edit, and monitor your data.",
		"msg:motd":            "Tip: press ? for help, / to filter keys",
	}
	for k, v := range data {
		must(rdb.Set(ctx, k, v, 0))
	}
	fmt.Printf("  strings:      %d keys\n", len(data))
}

func seedLists(ctx context.Context, rdb redis.Cmdable) {
	must(rdb.RPush(ctx, "queue:emails", "alice@example.com", "bob@example.com", "carol@example.com", "dave@example.com", "eve@example.com"))
	must(rdb.RPush(ctx, "queue:jobs", "resize-image:42", "send-invoice:108", "generate-report:7", "cleanup-cache:1"))
	must(rdb.RPush(ctx, "log:events", "user.login", "page.view", "item.purchase", "user.logout", "page.view", "user.login", "error.500"))
	fmt.Println("  lists:        3 keys")
}

func seedSets(ctx context.Context, rdb redis.Cmdable) {
	must(rdb.SAdd(ctx, "tags:popular", "go", "redis", "tui", "docker", "kubernetes", "terraform", "postgres"))
	must(rdb.SAdd(ctx, "online:users", "alice", "bob", "carol", "dave"))
	must(rdb.SAdd(ctx, "permissions:admin", "users.read", "users.write", "keys.delete", "config.edit", "server.restart"))
	fmt.Println("  sets:         3 keys")
}

func seedSortedSets(ctx context.Context, rdb redis.Cmdable) {
	leaderboard := []redis.Z{
		{Score: 9850, Member: "alice"},
		{Score: 8720, Member: "bob"},
		{Score: 7600, Member: "carol"},
		{Score: 6100, Member: "dave"},
		{Score: 5430, Member: "eve"},
		{Score: 4200, Member: "frank"},
		{Score: 3100, Member: "grace"},
		{Score: 1900, Member: "heidi"},
	}
	must(rdb.ZAdd(ctx, "leaderboard:weekly", leaderboard...))

	latency := []redis.Z{
		{Score: 12.4, Member: "us-east-1"},
		{Score: 18.7, Member: "us-west-2"},
		{Score: 45.2, Member: "eu-west-1"},
		{Score: 89.1, Member: "ap-southeast-1"},
		{Score: 120.5, Member: "sa-east-1"},
	}
	must(rdb.ZAdd(ctx, "metrics:latency-p99", latency...))
	fmt.Println("  sorted sets:  2 keys")
}

func seedHashes(ctx context.Context, rdb redis.Cmdable) {
	users := []struct {
		key    string
		fields map[string]any
	}{
		{"user:1001", map[string]any{"name": "Alice Johnson", "email": "alice@example.com", "role": "admin", "created": "2024-01-15", "logins": "142"}},
		{"user:1002", map[string]any{"name": "Bob Smith", "email": "bob@example.com", "role": "editor", "created": "2024-03-22", "logins": "87"}},
		{"user:1003", map[string]any{"name": "Carol Williams", "email": "carol@example.com", "role": "viewer", "created": "2024-06-10", "logins": "23"}},
	}
	for _, u := range users {
		must(rdb.HSet(ctx, u.key, u.fields))
	}

	must(rdb.HSet(ctx, "config:app", map[string]any{
		"max_connections": "100",
		"timeout_ms":      "5000",
		"retry_count":     "3",
		"log_level":       "info",
		"cache_ttl":       "3600",
	}))

	must(rdb.HSet(ctx, "session:abc123", map[string]any{
		"user_id":    "1001",
		"ip":         "192.168.1.42",
		"user_agent": "Mozilla/5.0",
		"created_at": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
	}))
	fmt.Println("  hashes:       5 keys")
}

func seedStreams(ctx context.Context, rdb redis.Cmdable) {
	events := []map[string]any{
		{"action": "login", "user": "alice", "ip": "10.0.0.1"},
		{"action": "page_view", "user": "alice", "path": "/dashboard"},
		{"action": "login", "user": "bob", "ip": "10.0.0.2"},
		{"action": "api_call", "user": "bob", "endpoint": "/api/keys", "status": "200"},
		{"action": "page_view", "user": "alice", "path": "/settings"},
		{"action": "logout", "user": "bob", "ip": "10.0.0.2"},
	}
	for _, e := range events {
		must(rdb.XAdd(ctx, &redis.XAddArgs{Stream: "stream:activity", Values: e}))
	}
	fmt.Println("  streams:      1 key")
}

func seedHyperLogLog(ctx context.Context, rdb redis.Cmdable) {
	visitors := make([]any, 20)
	for i := range visitors {
		visitors[i] = fmt.Sprintf("user:%d", 1000+i)
	}
	must(rdb.PFAdd(ctx, "hll:unique-visitors", visitors...))

	pages := make([]any, 15)
	for i := range pages {
		pages[i] = fmt.Sprintf("/page/%d", i+1)
	}
	must(rdb.PFAdd(ctx, "hll:unique-pages", pages...))
	fmt.Println("  hyperloglog:  2 keys")
}

func seedBitmaps(ctx context.Context, rdb redis.Cmdable) {
	// Simulate daily active users — set bits for user IDs that were active
	activeUsers := []int64{1, 5, 12, 27, 42, 100, 128, 200, 350, 500}
	for _, offset := range activeUsers {
		must(rdb.SetBit(ctx, "bitmap:user-activity:2024-01-15", offset, 1))
	}

	// Simulate feature flags — each bit position represents a feature
	featureFlags := []int64{0, 2, 5, 7} // features 0, 2, 5, 7 enabled
	for _, offset := range featureFlags {
		must(rdb.SetBit(ctx, "bitmap:feature-flags", offset, 1))
	}
	fmt.Println("  bitmaps:      2 keys")
}

func seedGeo(ctx context.Context, rdb redis.Cmdable) {
	offices := []*redis.GeoLocation{
		{Name: "New York", Longitude: -74.0060, Latitude: 40.7128},
		{Name: "London", Longitude: -0.1276, Latitude: 51.5074},
		{Name: "Tokyo", Longitude: 139.6917, Latitude: 35.6895},
		{Name: "Sydney", Longitude: 151.2093, Latitude: -33.8688},
		{Name: "San Francisco", Longitude: -122.4194, Latitude: 37.7749},
	}
	must(rdb.GeoAdd(ctx, "geo:offices", offices...))

	restaurants := []*redis.GeoLocation{
		{Name: "Pizza Palace", Longitude: -73.9857, Latitude: 40.7484},
		{Name: "Sushi Garden", Longitude: -73.9712, Latitude: 40.7614},
		{Name: "Taco Town", Longitude: -73.9934, Latitude: 40.7505},
		{Name: "Burger Barn", Longitude: -73.9787, Latitude: 40.7527},
		{Name: "Noodle House", Longitude: -73.9862, Latitude: 40.7559},
		{Name: "Café Central", Longitude: -73.9814, Latitude: 40.7681},
	}
	must(rdb.GeoAdd(ctx, "geo:restaurants", restaurants...))
	fmt.Println("  geo:          2 keys")
}

func seedTTLKeys(ctx context.Context, rdb redis.Cmdable) {
	ttls := []struct {
		key string
		val string
		ttl time.Duration
	}{
		{"cache:homepage", "<html>cached homepage</html>", 5 * time.Minute},
		{"cache:api:users", `[{"id":1},{"id":2}]`, 2 * time.Minute},
		{"ratelimit:10.0.0.1", "47", 60 * time.Second},
		{"lock:deploy", "worker-3", 30 * time.Second},
		{"temp:upload:xyz", "pending", 10 * time.Minute},
	}
	for _, t := range ttls {
		must(rdb.Set(ctx, t.key, t.val, t.ttl))
	}
	fmt.Printf("  ttl keys:     %d keys\n", len(ttls))
}

func seedNestedKeys(ctx context.Context, rdb redis.Cmdable) {
	keys := map[string]string{
		"api:v1:auth:token":      "abc-123-xyz",
		"api:v1:auth:refresh":    "ref-456",
		"api:v1:users:count":     "3042",
		"api:v1:users:active":    "891",
		"api:v2:auth:token":      "v2-token-789",
		"api:v2:users:count":     "3042",
		"db:primary:host":        "10.0.1.10",
		"db:primary:port":        "5432",
		"db:replica:host":        "10.0.1.11",
		"db:replica:port":        "5432",
		"service:web:status":     "running",
		"service:web:uptime":     "48h",
		"service:worker:status":  "running",
		"service:worker:pending": "12",
	}
	for k, v := range keys {
		must(rdb.Set(ctx, k, v, 0))
	}
	fmt.Printf("  nested keys:  %d keys\n", len(keys))
}

func seedJSONStrings(ctx context.Context, rdb redis.Cmdable) {
	jsons := map[string]string{
		"json:user-profile": `{"id":1001,"name":"Alice Johnson","preferences":{"theme":"dark","language":"en","notifications":true},"tags":["admin","beta"]}`,
		"json:api-response": fmt.Sprintf(`{"status":"ok","data":{"items":[{"id":1,"name":"Widget"},{"id":2,"name":"Gadget"}],"total":2},"timestamp":"%s"}`, time.Now().Format(time.RFC3339)),
		"json:error-log":    `{"level":"error","message":"connection timeout","code":504,"details":{"host":"10.0.1.5","port":6379,"retry":3}}`,
	}
	for k, v := range jsons {
		must(rdb.Set(ctx, k, v, 0))
	}
	fmt.Printf("  json strings: %d keys\n", len(jsons))
}

func seedProtobuf(ctx context.Context, rdb redis.Cmdable) {
	user := protoJoin(
		protoString(1, "Alice Johnson"),
		protoString(2, "alice@example.com"),
		protoVarint(3, 1001),
		protoNested(4, protoJoin(protoString(1, "dark"), protoString(2, "en"))),
	)
	must(rdb.Set(ctx, "pb:user-profile", user, 0))

	delivery := protoJoin(
		protoString(1, "rst-demo:DELIVERY"),
		protoString(3, "America/Toronto"),
		protoVarint(5, 42),
	)
	must(rdb.Set(ctx, "pb:delivery", delivery, 0))

	menu := protoJoin(
		protoString(1, "menu-id"),
		protoNested(4, protoJoin(
			protoString(1, "cat-1"),
			protoNested(2, protoJoin(protoString(1, "item-1"), protoString(2, "Margherita"))),
		)),
	)
	must(rdb.Set(ctx, "pb:s2:menu", s2.Encode(nil, menu), 0))

	session := protoJoin(
		protoString(1, "sess-abc123"),
		protoVarint(2, 1001),
		protoString(3, "192.168.1.42"),
		protoNested(4, protoJoin(protoString(1, "web"), protoVarint(2, 1))),
	)
	must(rdb.Set(ctx, "pb:s2:session", s2.Encode(nil, session), 0))

	fmt.Println("  protobuf:     4 keys (2 raw, 2 s2+protobuf)")
}

func protoJoin(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func protoString(num protowire.Number, s string) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendString(b, s)
	return b
}

func protoVarint(num protowire.Number, v uint64) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.VarintType)
	b = protowire.AppendVarint(b, v)
	return b
}

func protoNested(num protowire.Number, inner []byte) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendBytes(b, inner)
	return b
}

func hasJSONModule(ctx context.Context, rdb redis.Cmdable) bool {
	// Probe with a throwaway key to see if JSON.SET is available
	pipe := rdb.Pipeline()
	pipe.Do(ctx, "JSON.SET", "__redis_tui_probe__", "$", `"probe"`)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false
	}
	// Clean up probe key
	rdb.Del(ctx, "__redis_tui_probe__")
	return true
}

func seedJSON(ctx context.Context, rdb redis.Cmdable) {
	jsons := map[string]string{
		"rjson:config":          `{"max_retries":3,"timeout_ms":5000,"features":{"dark_mode":true,"beta":false},"allowed_origins":["https://example.com","https://app.example.com"]}`,
		"rjson:user-settings":   `{"user_id":1001,"theme":"dark","language":"en","notifications":{"email":true,"push":false,"sms":false},"dashboard":{"widgets":["metrics","logs","alerts"],"refresh_interval":30}}`,
		"rjson:product-catalog": `{"products":[{"id":1,"name":"Widget Pro","price":29.99,"tags":["electronics","gadget"]},{"id":2,"name":"Super Gadget","price":49.99,"tags":["electronics","premium"]},{"id":3,"name":"Basic Tool","price":9.99,"tags":["tools","basic"]}],"updated_at":"2025-01-15T10:30:00Z"}`,
	}
	pipe := rdb.Pipeline()
	for k, v := range jsons {
		pipe.Do(ctx, "JSON.SET", k, "$", v)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("  json (native): failed to seed (%v)", err)
		return
	}
	fmt.Printf("  json (native): %d keys\n", len(jsons))
}

// Overridable in tests.
var (
	logFatalf         = log.Fatalf
	makeClusterClient = newClusterClient
	checkJSON         = hasJSONModule
)

func must(cmd interface{ Err() error }) {
	if cmd.Err() != nil {
		logFatalf("redis command failed: %v", cmd.Err())
	}
}
