package types

import "time"

// KeyType represents Redis data types
type KeyType string

const (
	KeyTypeString      KeyType = "string"
	KeyTypeList        KeyType = "list"
	KeyTypeSet         KeyType = "set"
	KeyTypeZSet        KeyType = "zset"
	KeyTypeHash        KeyType = "hash"
	KeyTypeStream      KeyType = "stream"
	KeyTypeJSON        KeyType = "ReJSON-RL"
	KeyTypeHyperLogLog KeyType = "hyperloglog"
	KeyTypeBitmap      KeyType = "bitmap"
	KeyTypeGeo         KeyType = "geo"
	KeyTypeProtobuf    KeyType = "protobuf"
)

// RedisKey represents a key with metadata
type RedisKey struct {
	Key        string
	Type       KeyType
	TTL        time.Duration
	MemorySize int64
	IsFavorite bool
}

// RedisValue holds the value for any Redis type
type RedisValue struct {
	Type         KeyType
	StringValue  string
	ListValue    []string
	SetValue     []string
	ZSetValue    []ZSetMember
	HashValue    map[string]string
	StreamValue  []StreamEntry
	JSONValue    string
	HLLCount     int64       // cardinality for HyperLogLog (from PFCOUNT)
	GeoValue     []GeoMember // members with coordinates for Geo
	BitCount     int64       // bit count for Bitmap (from BITCOUNT)
	BitPositions []int64     // set bit positions for Bitmap display
	// DecodedValue is a human-readable rendering for binary formats (e.g. protobuf).
	DecodedValue string
	// DecodedFormat labels how DecodedValue was produced (e.g. "s2+protobuf").
	DecodedFormat string
	// RawSize / DecodedSize track original vs decompressed payload sizes.
	RawSize     int
	DecodedSize int
}

// GeoMember represents a geospatial member with coordinates
type GeoMember struct {
	Name      string
	Longitude float64
	Latitude  float64
}

// ZSetMember represents a sorted set member with score
type ZSetMember struct {
	Member string
	Score  float64
}

// StreamEntry represents a stream entry
type StreamEntry struct {
	ID     string
	Fields map[string]any
}

// ServerInfo holds Redis server information
type ServerInfo struct {
	Version       string
	Mode          string
	OS            string
	UsedMemory    string
	PeakMemory    string
	Clients       string
	TotalKeys     string
	Uptime        string
	ConnectedDB   int
	ClusterMode   bool
	ClusterInfo   string
	ReplicaInfo   string
	RDBLastSave   string
	AOFEnabled    bool
	MemFragRatio  string
	CPUUsage      string
	TotalCommands string
}

// SlowLogEntry represents a slow query log entry
type SlowLogEntry struct {
	ID         int64
	Timestamp  time.Time
	Duration   time.Duration
	Command    string
	ClientAddr string
	ClientName string
}

// PubSubChannel represents a pub/sub channel
type PubSubChannel struct {
	Name        string
	Subscribers int64
}

// PubSubMessage represents a received pub/sub message
type PubSubMessage struct {
	Channel string
	Message string
	Time    time.Time
}

// ClientInfo represents connected client information
type ClientInfo struct {
	ID       int64
	Addr     string
	Name     string
	Age      time.Duration
	Idle     time.Duration
	Flags    string
	DB       int
	Cmd      string
	SubCount int
}

// ClusterNode represents a Redis cluster node
type ClusterNode struct {
	ID         string
	Addr       string
	Flags      string
	Role       string
	Master     string
	PingSent   int64
	PongRecv   int64
	ConfigEpoc int64
	LinkState  string
	Slots      string
	SlotStart  int
	SlotEnd    int
}

// MemoryStats holds memory statistics
type MemoryStats struct {
	TotalMemory        int64
	UsedMemory         int64
	PeakMemory         int64
	FragmentedBytes    int64
	FragRatio          float64
	FragmentationRatio float64
	RSS                string
	LuaMemory          string
	ByType             map[KeyType]int64
	TopKeys            []KeyMemory
}

// KeyMemory holds memory info for a specific key
type KeyMemory struct {
	Key    string
	Type   KeyType
	Memory int64
	Bytes  int64
}

// ValueHistoryEntry stores a previous value for undo
type ValueHistoryEntry struct {
	Key       string
	Value     RedisValue
	Timestamp time.Time
	Action    string // "set", "delete", "modify"
}

// KeyspaceEvent represents a keyspace notification
type KeyspaceEvent struct {
	Timestamp time.Time
	DB        int
	Event     string
	Key       string
}

// LiveMetricsData holds real-time metrics data points
type LiveMetricsData struct {
	Timestamp        time.Time
	OpsPerSec        float64
	UsedMemoryBytes  int64
	ConnectedClients int64
	BlockedClients   int64
	KeyspaceHits     int64
	KeyspaceMisses   int64
	ExpiredKeys      int64
	EvictedKeys      int64
	InputKbps        float64
	OutputKbps       float64
	UsedCPUSys       float64
	UsedCPUUser      float64
	TotalConnections int64
	RejectedConns    int64
}

// RedisConfigParam represents a Redis configuration parameter for ordered display
type RedisConfigParam struct {
	Name  string
	Value string
}

// LiveMetrics holds historical metrics for charting
type LiveMetrics struct {
	DataPoints      []LiveMetricsData
	MaxDataPoints   int
	RefreshInterval time.Duration
}
