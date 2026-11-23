package testutil

import (
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

// MockConfigClient implements the complete service.ConfigService interface for testing.
type MockConfigClient struct {
	// Configurable return values
	CloseError              error
	ListConnectionsResult   []types.Connection
	ListConnectionsError    error
	AddConnectionResult     types.Connection
	AddConnectionError      error
	UpdateConnectionResult  types.Connection
	UpdateConnectionError   error
	DeleteConnectionError   error
	AddFavoriteResult       types.Favorite
	AddFavoriteError        error
	RemoveFavoriteError     error
	ListFavoritesResult     []types.Favorite
	IsFavoriteResult        bool
	ListRecentKeysResult    []types.RecentKey
	GetValueHistoryResult   []types.ValueHistoryEntry
	ListTemplatesResult     []types.KeyTemplate
	AddTemplateError        error
	DeleteTemplateError     error
	ListGroupsResult        []types.ConnectionGroup
	AddGroupError           error
	AddConnectionToGrpError error
	RemoveConnFromGrpError  error
	KeyBindingsResult       types.KeyBindings
	SetKeyBindingsError     error
	ResetKeyBindingsError   error
	TreeSeparatorResult     string
	SetTreeSeparatorError   error
	WatchIntervalResult     time.Duration

	// Call counters for void methods that otherwise have no observable effect.
	AddRecentKeyCalls      int
	ClearRecentKeysCalls   int
	AddValueHistoryCalls   int
	ClearValueHistoryCalls int
}

// NewMockConfigClient creates a new fully-mocked Config client.
func NewMockConfigClient() *MockConfigClient {
	return &MockConfigClient{
		TreeSeparatorResult: ":",
		WatchIntervalResult: time.Second,
	}
}

func (m *MockConfigClient) Close() error { return m.CloseError }
func (m *MockConfigClient) ListConnections() ([]types.Connection, error) {
	return m.ListConnectionsResult, m.ListConnectionsError
}

func (m *MockConfigClient) AddConnection(_ types.Connection) (types.Connection, error) {
	return m.AddConnectionResult, m.AddConnectionError
}

func (m *MockConfigClient) UpdateConnection(_ types.Connection) (types.Connection, error) {
	return m.UpdateConnectionResult, m.UpdateConnectionError
}
func (m *MockConfigClient) DeleteConnection(_ int64) error { return m.DeleteConnectionError }
func (m *MockConfigClient) AddFavorite(_ int64, _, _ string) (types.Favorite, error) {
	return m.AddFavoriteResult, m.AddFavoriteError
}
func (m *MockConfigClient) RemoveFavorite(_ int64, _ string) error { return m.RemoveFavoriteError }
func (m *MockConfigClient) ListFavorites(_ int64) []types.Favorite { return m.ListFavoritesResult }
func (m *MockConfigClient) IsFavorite(_ int64, _ string) bool      { return m.IsFavoriteResult }
func (m *MockConfigClient) AddRecentKey(_ int64, _ string, _ types.KeyType) {
	m.AddRecentKeyCalls++
}
func (m *MockConfigClient) ListRecentKeys(_ int64) []types.RecentKey { return m.ListRecentKeysResult }
func (m *MockConfigClient) ClearRecentKeys(_ int64)                  { m.ClearRecentKeysCalls++ }
func (m *MockConfigClient) AddValueHistory(_ string, _ types.RedisValue, _ string) {
	m.AddValueHistoryCalls++
}
func (m *MockConfigClient) GetValueHistory(_ string) []types.ValueHistoryEntry {
	return m.GetValueHistoryResult
}
func (m *MockConfigClient) ClearValueHistory()                    { m.ClearValueHistoryCalls++ }
func (m *MockConfigClient) ListTemplates() []types.KeyTemplate    { return m.ListTemplatesResult }
func (m *MockConfigClient) AddTemplate(_ types.KeyTemplate) error { return m.AddTemplateError }
func (m *MockConfigClient) DeleteTemplate(_ string) error         { return m.DeleteTemplateError }
func (m *MockConfigClient) ListGroups() []types.ConnectionGroup   { return m.ListGroupsResult }
func (m *MockConfigClient) AddGroup(_, _ string) error            { return m.AddGroupError }
func (m *MockConfigClient) AddConnectionToGroup(_ string, _ int64) error {
	return m.AddConnectionToGrpError
}
func (m *MockConfigClient) RemoveConnectionFromGroup(_ string, _ int64) error {
	return m.RemoveConnFromGrpError
}
func (m *MockConfigClient) GetKeyBindings() types.KeyBindings        { return m.KeyBindingsResult }
func (m *MockConfigClient) SetKeyBindings(_ types.KeyBindings) error { return m.SetKeyBindingsError }
func (m *MockConfigClient) ResetKeyBindings() error                  { return m.ResetKeyBindingsError }
func (m *MockConfigClient) GetTreeSeparator() string                 { return m.TreeSeparatorResult }
func (m *MockConfigClient) SetTreeSeparator(_ string) error          { return m.SetTreeSeparatorError }
func (m *MockConfigClient) GetWatchInterval() time.Duration          { return m.WatchIntervalResult }
