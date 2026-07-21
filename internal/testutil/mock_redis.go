package testutil

import (
	"errors"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

// ErrMockNotConnected is returned when attempting to use an unconnected mock client.
var ErrMockNotConnected = errors.New("mock redis client not connected")

// MockRedisClient is a mock implementation of Redis client operations for testing.
type MockRedisClient struct {
	connected bool
	keys      map[string]types.RedisValue
	keyTypes  map[string]types.KeyType
	keyTTLs   map[string]time.Duration

	// Configurable behaviors
	ConnectError    error
	DisconnectError error
	ScanError       error
	GetError        error
	SetError        error
	DeleteError     error
}

// NewMockRedisClient creates a new mock Redis client.
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		keys:     make(map[string]types.RedisValue),
		keyTypes: make(map[string]types.KeyType),
		keyTTLs:  make(map[string]time.Duration),
	}
}

// Connect simulates connecting to Redis.
func (m *MockRedisClient) Connect(conn types.Connection) error {
	if m.ConnectError != nil {
		return m.ConnectError
	}
	m.connected = true
	return nil
}

// Disconnect simulates disconnecting from Redis.
func (m *MockRedisClient) Disconnect() error {
	if m.DisconnectError != nil {
		return m.DisconnectError
	}
	m.connected = false
	return nil
}

// IsConnected returns whether the mock client is connected.
func (m *MockRedisClient) IsConnected() bool {
	return m.connected
}

// SetKey sets a key-value pair in the mock store.
func (m *MockRedisClient) SetKey(key string, value types.RedisValue, keyType types.KeyType, ttl time.Duration) {
	m.keys[key] = value
	m.keyTypes[key] = keyType
	m.keyTTLs[key] = ttl
}

// GetValue retrieves a value from the mock store.
func (m *MockRedisClient) GetValue(key string) (types.RedisValue, error) {
	if !m.connected {
		return types.RedisValue{}, ErrMockNotConnected
	}
	if m.GetError != nil {
		return types.RedisValue{}, m.GetError
	}
	value, exists := m.keys[key]
	if !exists {
		return types.RedisValue{}, errors.New("key not found")
	}
	return value, nil
}

// GetValuePreview returns the stored value like GetValue; the mock does not
// simulate truncation.
func (m *MockRedisClient) GetValuePreview(key string) (types.RedisValue, error) {
	return m.GetValue(key)
}

// ScanKeys returns keys matching a pattern from the mock store.
func (m *MockRedisClient) ScanKeys(pattern string, cursor uint64, count int64) ([]types.RedisKey, uint64, error) {
	if !m.connected {
		return nil, 0, ErrMockNotConnected
	}
	if m.ScanError != nil {
		return nil, 0, m.ScanError
	}

	var result []types.RedisKey
	for key, keyType := range m.keyTypes {
		if pattern == "*" || matchPattern(pattern, key) {
			result = append(result, types.RedisKey{
				Key:  key,
				Type: keyType,
				TTL:  m.keyTTLs[key],
			})
		}
	}
	return result, 0, nil
}

// DeleteKey removes a key from the mock store.
func (m *MockRedisClient) DeleteKey(key string) error {
	if !m.connected {
		return ErrMockNotConnected
	}
	if m.DeleteError != nil {
		return m.DeleteError
	}
	delete(m.keys, key)
	delete(m.keyTypes, key)
	delete(m.keyTTLs, key)
	return nil
}

// GetTotalKeys returns the total number of keys in the mock store.
func (m *MockRedisClient) GetTotalKeys() int64 {
	return int64(len(m.keys))
}

// Reset clears the mock store and resets connection state.
func (m *MockRedisClient) Reset() {
	m.connected = false
	m.keys = make(map[string]types.RedisValue)
	m.keyTypes = make(map[string]types.KeyType)
	m.keyTTLs = make(map[string]time.Duration)
	m.ConnectError = nil
	m.DisconnectError = nil
	m.ScanError = nil
	m.GetError = nil
	m.SetError = nil
	m.DeleteError = nil
}

// matchPattern is a simple pattern matcher for testing.
// Only supports * as wildcard at the end.
func matchPattern(pattern, str string) bool {
	if pattern == "*" {
		return true
	}
	// Simple prefix matching for patterns like "user:*"
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(str) >= len(prefix) && str[:len(prefix)] == prefix
	}
	return pattern == str
}
