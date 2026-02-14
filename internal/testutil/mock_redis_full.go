package testutil

import (
	"crypto/tls"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// FullMockRedisClient implements the complete service.RedisService interface for testing.
type FullMockRedisClient struct {
	// Embed MockRedisClient for Connect/Disconnect/GetValue/ScanKeys/DeleteKey/GetTotalKeys
	*MockRedisClient

	// Configurable return values
	IsClusterResult       bool
	TestConnectionLatency time.Duration
	ServerInfo            types.ServerInfo
	MemStats              types.MemoryStats
	MemUsageResult        int64
	SlowLogEntries        []types.SlowLogEntry
	ClientListResult      []types.ClientInfo
	ClusterNodesResult    []types.ClusterNode
	ClusterInfoResult     string
	EvalResult            interface{}
	PublishResult         int64
	PubSubChannelsResult  []string
	ExportResult          map[string]interface{}
	ImportResult          int
	BulkDeleteResult      int
	BatchTTLResult        int
	SearchByValueResult   []types.RedisKey
	RegexSearchResult     []types.RedisKey
	FuzzySearchResult     []types.RedisKey
	CompareValue1         types.RedisValue
	CompareValue2         types.RedisValue
	KeyPrefixesResult     []string
	XAddResult            string

	// Configurable errors (one per method)
	ConnectClusterError    error
	ConnectWithTLSError    error
	TestConnectionError    error
	SetStringError         error
	SetTTLError            error
	BatchSetTTLError       error
	RPushError             error
	LSetError              error
	LRemError              error
	SAddError              error
	SRemError              error
	ZAddError              error
	ZRemError              error
	HSetError              error
	HDelError              error
	XAddError              error
	XDelError              error
	SelectDBError          error
	FlushDBError           error
	ServerInfoError        error
	MemStatsError          error
	MemUsageError          error
	SlowLogError           error
	ClientListError        error
	ClusterNodesError      error
	ClusterInfoError       error
	EvalError              error
	PublishError           error
	PubSubChannelsError    error
	ExportError            error
	ImportError            error
	RenameError            error
	CopyError              error
	DeleteKeysError        error
	BulkDeleteError        error
	SearchByValueError     error
	RegexSearchError       error
	FuzzySearchError       error
	CompareKeysError       error
	KeyPrefixesError       error
	SubscribeKeyspaceError error
	UnsubscribeKSError     error

	// Call tracking
	Calls []string
}

// NewFullMockRedisClient creates a new fully-mocked Redis client.
func NewFullMockRedisClient() *FullMockRedisClient {
	return &FullMockRedisClient{
		MockRedisClient: NewMockRedisClient(),
	}
}

// Connection management

func (m *FullMockRedisClient) Connect(host string, port int, password string, db int) error {
	m.Calls = append(m.Calls, "Connect")
	return m.MockRedisClient.Connect(host, port, password, db)
}

func (m *FullMockRedisClient) ConnectWithTLS(_ string, _ int, _ string, _ int, _ *tls.Config) error {
	m.Calls = append(m.Calls, "ConnectWithTLS")
	if m.ConnectWithTLSError != nil {
		return m.ConnectWithTLSError
	}
	return m.MockRedisClient.Connect("", 0, "", 0)
}

func (m *FullMockRedisClient) ConnectCluster(_ []string, _ string) error {
	m.Calls = append(m.Calls, "ConnectCluster")
	if m.ConnectClusterError != nil {
		return m.ConnectClusterError
	}
	return m.MockRedisClient.Connect("", 0, "", 0)
}

func (m *FullMockRedisClient) IsCluster() bool {
	return m.IsClusterResult
}

func (m *FullMockRedisClient) TestConnection(_ string, _ int, _ string, _ int) (time.Duration, error) {
	m.Calls = append(m.Calls, "TestConnection")
	return m.TestConnectionLatency, m.TestConnectionError
}

// Key operations

func (m *FullMockRedisClient) ScanKeysWithRegex(_ string, _ int) ([]types.RedisKey, error) {
	m.Calls = append(m.Calls, "ScanKeysWithRegex")
	return m.RegexSearchResult, m.RegexSearchError
}

func (m *FullMockRedisClient) FuzzySearchKeys(_ string, _ int) ([]types.RedisKey, error) {
	m.Calls = append(m.Calls, "FuzzySearchKeys")
	return m.FuzzySearchResult, m.FuzzySearchError
}

func (m *FullMockRedisClient) DeleteKeys(_ ...string) (int64, error) {
	m.Calls = append(m.Calls, "DeleteKeys")
	return 0, m.DeleteKeysError
}

func (m *FullMockRedisClient) BulkDelete(_ string) (int, error) {
	m.Calls = append(m.Calls, "BulkDelete")
	return m.BulkDeleteResult, m.BulkDeleteError
}

func (m *FullMockRedisClient) Rename(_, _ string) error {
	m.Calls = append(m.Calls, "Rename")
	return m.RenameError
}

func (m *FullMockRedisClient) Copy(_, _ string, _ bool) error {
	m.Calls = append(m.Calls, "Copy")
	return m.CopyError
}

func (m *FullMockRedisClient) SearchByValue(_, _ string, _ int) ([]types.RedisKey, error) {
	m.Calls = append(m.Calls, "SearchByValue")
	return m.SearchByValueResult, m.SearchByValueError
}

func (m *FullMockRedisClient) CompareKeys(_, _ string) (types.RedisValue, types.RedisValue, error) {
	m.Calls = append(m.Calls, "CompareKeys")
	return m.CompareValue1, m.CompareValue2, m.CompareKeysError
}

func (m *FullMockRedisClient) GetKeyPrefixes(_ string, _ int) ([]string, error) {
	m.Calls = append(m.Calls, "GetKeyPrefixes")
	return m.KeyPrefixesResult, m.KeyPrefixesError
}

// String operations

func (m *FullMockRedisClient) SetString(_, _ string, _ time.Duration) error {
	m.Calls = append(m.Calls, "SetString")
	return m.SetStringError
}

// TTL operations

func (m *FullMockRedisClient) SetTTL(_ string, _ time.Duration) error {
	m.Calls = append(m.Calls, "SetTTL")
	return m.SetTTLError
}

func (m *FullMockRedisClient) BatchSetTTL(_ string, _ time.Duration) (int, error) {
	m.Calls = append(m.Calls, "BatchSetTTL")
	return m.BatchTTLResult, m.BatchSetTTLError
}

// List operations

func (m *FullMockRedisClient) RPush(_ string, _ ...string) error {
	m.Calls = append(m.Calls, "RPush")
	return m.RPushError
}

func (m *FullMockRedisClient) LSet(_ string, _ int64, _ string) error {
	m.Calls = append(m.Calls, "LSet")
	return m.LSetError
}

func (m *FullMockRedisClient) LRem(_ string, _ int64, _ string) error {
	m.Calls = append(m.Calls, "LRem")
	return m.LRemError
}

// Set operations

func (m *FullMockRedisClient) SAdd(_ string, _ ...string) error {
	m.Calls = append(m.Calls, "SAdd")
	return m.SAddError
}

func (m *FullMockRedisClient) SRem(_ string, _ ...string) error {
	m.Calls = append(m.Calls, "SRem")
	return m.SRemError
}

// Sorted set operations

func (m *FullMockRedisClient) ZAdd(_ string, _ float64, _ string) error {
	m.Calls = append(m.Calls, "ZAdd")
	return m.ZAddError
}

func (m *FullMockRedisClient) ZRem(_ string, _ ...string) error {
	m.Calls = append(m.Calls, "ZRem")
	return m.ZRemError
}

// Hash operations

func (m *FullMockRedisClient) HSet(_, _, _ string) error {
	m.Calls = append(m.Calls, "HSet")
	return m.HSetError
}

func (m *FullMockRedisClient) HDel(_ string, _ ...string) error {
	m.Calls = append(m.Calls, "HDel")
	return m.HDelError
}

// Stream operations

func (m *FullMockRedisClient) XAdd(_ string, _ map[string]interface{}) (string, error) {
	m.Calls = append(m.Calls, "XAdd")
	return m.XAddResult, m.XAddError
}

func (m *FullMockRedisClient) XDel(_ string, _ ...string) error {
	m.Calls = append(m.Calls, "XDel")
	return m.XDelError
}

// Database operations

func (m *FullMockRedisClient) SelectDB(_ int) error {
	m.Calls = append(m.Calls, "SelectDB")
	return m.SelectDBError
}

func (m *FullMockRedisClient) FlushDB() error {
	m.Calls = append(m.Calls, "FlushDB")
	return m.FlushDBError
}

// Server info and monitoring

func (m *FullMockRedisClient) GetServerInfo() (types.ServerInfo, error) {
	m.Calls = append(m.Calls, "GetServerInfo")
	return m.ServerInfo, m.ServerInfoError
}

func (m *FullMockRedisClient) GetMemoryStats() (types.MemoryStats, error) {
	m.Calls = append(m.Calls, "GetMemoryStats")
	return m.MemStats, m.MemStatsError
}

func (m *FullMockRedisClient) MemoryUsage(_ string) (int64, error) {
	m.Calls = append(m.Calls, "MemoryUsage")
	return m.MemUsageResult, m.MemUsageError
}

func (m *FullMockRedisClient) SlowLogGet(_ int64) ([]types.SlowLogEntry, error) {
	m.Calls = append(m.Calls, "SlowLogGet")
	return m.SlowLogEntries, m.SlowLogError
}

func (m *FullMockRedisClient) ClientList() ([]types.ClientInfo, error) {
	m.Calls = append(m.Calls, "ClientList")
	return m.ClientListResult, m.ClientListError
}

// Cluster operations

func (m *FullMockRedisClient) ClusterNodes() ([]types.ClusterNode, error) {
	m.Calls = append(m.Calls, "ClusterNodes")
	return m.ClusterNodesResult, m.ClusterNodesError
}

func (m *FullMockRedisClient) ClusterInfo() (string, error) {
	m.Calls = append(m.Calls, "ClusterInfo")
	return m.ClusterInfoResult, m.ClusterInfoError
}

// Scripting

func (m *FullMockRedisClient) Eval(_ string, _ []string, _ ...interface{}) (interface{}, error) {
	m.Calls = append(m.Calls, "Eval")
	return m.EvalResult, m.EvalError
}

// Pub/Sub

func (m *FullMockRedisClient) Publish(_, _ string) (int64, error) {
	m.Calls = append(m.Calls, "Publish")
	return m.PublishResult, m.PublishError
}

func (m *FullMockRedisClient) Subscribe(_ string) *redis.PubSub {
	m.Calls = append(m.Calls, "Subscribe")
	return nil
}

func (m *FullMockRedisClient) PubSubChannels(_ string) ([]string, error) {
	m.Calls = append(m.Calls, "PubSubChannels")
	return m.PubSubChannelsResult, m.PubSubChannelsError
}

// Keyspace events

func (m *FullMockRedisClient) SubscribeKeyspace(_ string, _ func(types.KeyspaceEvent)) error {
	m.Calls = append(m.Calls, "SubscribeKeyspace")
	return m.SubscribeKeyspaceError
}

func (m *FullMockRedisClient) UnsubscribeKeyspace() error {
	m.Calls = append(m.Calls, "UnsubscribeKeyspace")
	return m.UnsubscribeKSError
}

// Import/Export

func (m *FullMockRedisClient) ExportKeys(_ string) (map[string]interface{}, error) {
	m.Calls = append(m.Calls, "ExportKeys")
	return m.ExportResult, m.ExportError
}

func (m *FullMockRedisClient) ImportKeys(_ map[string]interface{}) (int, error) {
	m.Calls = append(m.Calls, "ImportKeys")
	return m.ImportResult, m.ImportError
}
