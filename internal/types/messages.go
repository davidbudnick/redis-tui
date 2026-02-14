package types

import "time"

// Messages for Bubble Tea updates

type ConnectionsLoadedMsg struct {
	Connections []Connection
	Err         error
}

type ConnectionAddedMsg struct {
	Connection Connection
	Err        error
}

type ConnectionUpdatedMsg struct {
	Connection Connection
	Err        error
}

type ConnectionDeletedMsg struct {
	ID  int64
	Err error
}

type ConnectedMsg struct {
	Err error
}

// AutoConnectMsg triggers an automatic connection from CLI flags.
type AutoConnectMsg struct {
	Connection Connection
}

type DisconnectedMsg struct{}

type KeysLoadedMsg struct {
	Keys      []RedisKey
	Cursor    uint64
	TotalKeys int64
	Err       error
}

type KeyValueLoadedMsg struct {
	Key   string
	Value RedisValue
	Err   error
}

type KeyDeletedMsg struct {
	Key string
	Err error
}

type KeySetMsg struct {
	Key string
	Err error
}

type ServerInfoLoadedMsg struct {
	Info ServerInfo
	Err  error
}

type TTLSetMsg struct {
	Key string
	TTL time.Duration
	Err error
}

type FlushDBMsg struct {
	Err error
}

type ValueEditedMsg struct {
	Key string
	Err error
}

type ItemAddedToCollectionMsg struct {
	Key string
	Err error
}

type ItemRemovedFromCollectionMsg struct {
	Key string
	Err error
}

type KeyRenamedMsg struct {
	OldKey string
	NewKey string
	Err    error
}

type KeyCopiedMsg struct {
	SourceKey string
	DestKey   string
	Err       error
}

type DBSwitchedMsg struct {
	DB  int
	Err error
}

type SlowLogLoadedMsg struct {
	Entries []SlowLogEntry
	Err     error
}

type LuaScriptResultMsg struct {
	Result interface{}
	Err    error
}

type ConnectionTestMsg struct {
	Success bool
	Err     error
	Latency time.Duration
}

type MemoryUsageMsg struct {
	Key   string
	Bytes int64
	Err   error
}

type ExportCompleteMsg struct {
	Filename string
	KeyCount int
	Err      error
}

type ImportCompleteMsg struct {
	Filename string
	KeyCount int
	Err      error
}

type PublishResultMsg struct {
	Channel   string
	Receivers int64
	Err       error
}

type TickMsg struct{}

type WatchTickMsg struct{}

type TTLRefreshMsg struct {
	Keys []RedisKey
	Err  error
}

// New messages for additional features

type BulkDeleteMsg struct {
	Pattern string
	Deleted int
	Err     error
}

type FavoritesLoadedMsg struct {
	Favorites []Favorite
	Err       error
}

type FavoriteAddedMsg struct {
	Favorite Favorite
	Err      error
}

type FavoriteRemovedMsg struct {
	Key string
	Err error
}

type RecentKeysLoadedMsg struct {
	Keys []RecentKey
	Err  error
}

type ClientListLoadedMsg struct {
	Clients []ClientInfo
	Err     error
}

type MemoryStatsLoadedMsg struct {
	Stats MemoryStats
	Err   error
}

type ClusterInfoLoadedMsg struct {
	Nodes []ClusterNode
	Info  string
	Err   error
}

// ClusterNodesLoadedMsg is dispatched after connect to populate node count (without navigating to cluster screen).
type ClusterNodesLoadedMsg struct {
	Nodes []ClusterNode
	Err   error
}

type BatchTTLSetMsg struct {
	Pattern string
	Count   int
	TTL     time.Duration
	Err     error
}

type TemplatesLoadedMsg struct {
	Templates []KeyTemplate
	Err       error
}

type ValueHistoryMsg struct {
	History []ValueHistoryEntry
	Err     error
}

type ValueRestoredMsg struct {
	Key string
	Err error
}

type KeyspaceEventMsg struct {
	Event KeyspaceEvent
}

type KeyspaceSubscribedMsg struct {
	Pattern string
	Err     error
}

type ClipboardCopiedMsg struct {
	Content string
	Err     error
}

type JSONPathResultMsg struct {
	Result interface{}
	Err    error
}

type SSHTunnelConnectedMsg struct {
	LocalPort int
	Err       error
}

type FuzzySearchResultMsg struct {
	Keys []RedisKey
	Err  error
}

type RegexSearchResultMsg struct {
	Keys []RedisKey
	Err  error
}

type CompareKeysResultMsg struct {
	Key1Value RedisValue
	Key2Value RedisValue
	Diff      string
	Err       error
}

type TreeNodeExpandedMsg struct {
	Prefix   string
	Children []string
	Err      error
}

// Additional messages needed

type TreeViewLoadedMsg struct {
	Nodes []TreeNode
	Err   error
}

type SearchResultsMsg struct {
	Keys []RedisKey
	Err  error
}

type WatchKeyUpdateMsg struct {
	Key       string
	Value     string
	Timestamp time.Time
	Err       error
}

type KeyComparisonMsg struct {
	Result *KeyComparison
	Err    error
}

type ValueHistoryLoadedMsg struct {
	History []ValueHistoryEntry
	Err     error
}

type CopyToClipboardMsg struct {
	Content string
	Err     error
}

type GroupsLoadedMsg struct {
	Groups []ConnectionGroup
	Err    error
}

// KeyPreviewLoadedMsg is sent when a key preview value is loaded
type KeyPreviewLoadedMsg struct {
	Key   string
	Value RedisValue
	Err   error
}

// LiveMetricsMsg is sent when live metrics are fetched
type LiveMetricsMsg struct {
	Data LiveMetricsData
	Err  error
}

// LiveMetricsTickMsg triggers a metrics refresh
type LiveMetricsTickMsg struct{}

// KeyComparison holds comparison result between two keys
type KeyComparison struct {
	Key1        string
	Key2        string
	Equal       bool
	Differences []string
}

// EditorSaveMsg is sent when user saves in vim editor (:w)
type EditorSaveMsg struct {
	Content string
}

// EditorQuitMsg is sent when user quits vim editor (:q)
type EditorQuitMsg struct{}
