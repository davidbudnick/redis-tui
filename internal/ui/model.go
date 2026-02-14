package ui

import (
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/davidbudnick/redis-tui/internal/cmd"
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/kujtimiihoxha/vimtea"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Screen            types.Screen
	Connections       []types.Connection
	SelectedConnIdx   int
	ConnInputs        []textinput.Model
	ConnFocusIdx      int
	EditingConnection *types.Connection
	CurrentConn       *types.Connection
	Keys              []types.RedisKey
	SelectedKeyIdx    int
	KeyCursor         uint64
	KeyPattern        string
	PatternInput      textinput.Model
	CurrentKey        *types.RedisKey
	CurrentValue      types.RedisValue
	AddKeyInputs      []textinput.Model
	AddKeyFocusIdx    int
	AddKeyType        types.KeyType
	TTLInput          textinput.Model
	ServerInfo        types.ServerInfo
	TotalKeys         int64
	Width             int
	Height            int
	Err               error
	StatusMsg         string
	Loading           bool
	ConfirmType       string
	ConfirmData       interface{}
	Logs              *[]string
	SendFunc          *func(tea.Msg)
	PendingSelectKey  string

	// New fields for additional features
	VimEditor vimtea.Editor
	EditingIndex       int
	EditingField       string
	AddCollectionInput []textinput.Model
	AddCollFocusIdx    int
	RenameInput        textinput.Model
	CopyInput          textinput.Model
	SearchValueInput   textinput.Model
	ExportInput        textinput.Model
	ImportInput        textinput.Model
	LuaScriptInput     textinput.Model
	LuaResult          string
	PubSubInput        []textinput.Model
	PubSubFocusIdx     int
	PubSubMessages     []types.PubSubMessage
	SlowLogEntries     []types.SlowLogEntry
	MemoryUsage        int64
	SelectedItemIdx    int
	SortBy             string
	SortAsc            bool
	DBSwitchInput      textinput.Model
	TestConnResult     string
	LogCursor          int
	ShowingLogDetail   bool

	// Favorites and recent keys
	Favorites         []types.Favorite
	RecentKeys        []types.RecentKey
	SelectedFavIdx    int
	SelectedRecentIdx int

	// Tree view
	TreeNodes       []types.TreeNode
	TreeExpanded    map[string]bool
	TreeSeparator   string
	SelectedTreeIdx int

	// Bulk operations
	BulkDeleteInput   textinput.Model
	BulkDeletePreview []string
	BulkDeleteCount   int
	SelectedBulkKeys  map[string]bool

	// Batch TTL
	BatchTTLInput   textinput.Model
	BatchTTLPattern textinput.Model
	BatchTTLPreview []string

	// Search
	RegexSearchInput textinput.Model
	FuzzySearchInput textinput.Model
	SearchResults    []types.RedisKey

	// Watch mode
	WatchActive     bool
	WatchKey        string
	WatchValue      string
	WatchLastUpdate time.Time
	WatchInterval   time.Duration

	// Client list and memory stats
	ClientList        []types.ClientInfo
	MemoryStats       *types.MemoryStats
	SelectedClientIdx int

	// Cluster mode
	ClusterNodes    []types.ClusterNode
	ClusterEnabled  bool
	SelectedNodeIdx int
	ConnClusterMode bool

	// Compare keys
	CompareKey1Input textinput.Model
	CompareKey2Input textinput.Model
	CompareResult    *types.KeyComparison
	CompareFocusIdx  int

	// Key templates
	Templates           []types.KeyTemplate
	SelectedTemplateIdx int
	TemplateInputs      []textinput.Model
	TemplateFocusIdx    int

	// JSON path query
	JSONPathInput  textinput.Model
	JSONPathResult string

	// Keybindings
	KeyBindings types.KeyBindings

	// Value history
	ValueHistory       []types.ValueHistoryEntry
	SelectedHistoryIdx int

	// Keyspace events
	KeyspaceEvents    []types.KeyspaceEvent
	KeyspaceSubActive bool
	KeyspacePattern   string

	// Connection groups
	ConnectionGroups []types.ConnectionGroup
	SelectedGroupIdx int

	// Expiring keys alerts
	ExpiringKeys    []types.RedisKey
	ExpiryThreshold int64 // seconds

	// Last tick time for accurate TTL counting
	LastTickTime time.Time

	// Key preview (shown in keys list)
	PreviewKey    string
	PreviewValue  types.RedisValue
	PreviewScroll int
	DetailScroll  int
	DetailLines   []string

	// Live metrics dashboard
	LiveMetrics       *types.LiveMetrics
	LiveMetricsActive bool

	// Connection error (for prominent display)
	ConnectionError string

	// CLI auto-connect (set when --host flag is provided)
	CLIConnection *types.Connection

	// Lazy initialization flag
	inputsInitialized bool
}

func NewModel() Model {
	return Model{
		Screen:             types.ScreenConnections,
		Connections:        []types.Connection{},
		ConnInputs:         createConnectionInputs(),
		Keys:               []types.RedisKey{},
		AddKeyInputs:       createAddKeyInputs(),
		AddCollectionInput: createAddCollectionInputs(),
		PubSubInput:        createPubSubInputs(),
		AddKeyType:         types.KeyTypeString,
		SortBy:             "name",
		SortAsc:            true,
		TreeExpanded:       make(map[string]bool),
		TreeSeparator:      ":",
		SelectedBulkKeys:   make(map[string]bool),
		WatchInterval:      time.Second * 2,
		KeyBindings:        types.DefaultKeyBindings(),
		ExpiryThreshold:    300,
		PatternInput:       createTextInput("Filter pattern...", 40),
		TTLInput:           createTextInput("TTL in seconds (-1 to remove)", 30),
		RenameInput:        createTextInput("New key name", 40),
		CopyInput:          createTextInput("New key name for copy", 40),
		SearchValueInput:   createTextInput("Search in values...", 40),
		ExportInput:        createTextInput("Export filename", 40),
		ImportInput:        createTextInput("Import filename", 40),
		LuaScriptInput:     createTextInput("Lua script", 60),
		DBSwitchInput:      createTextInput("Database number (0-15)", 30),
		BulkDeleteInput:    createTextInput("Pattern to delete (e.g., user:*)", 40),
		BatchTTLInput:      createTextInput("TTL in seconds", 30),
		BatchTTLPattern:    createTextInput("Key pattern", 40),
		RegexSearchInput:   createTextInput("Regex pattern", 40),
		FuzzySearchInput:   createTextInput("Fuzzy search...", 40),
		CompareKey1Input:   createTextInput("First key", 40),
		CompareKey2Input:   createTextInput("Second key", 40),
		JSONPathInput:      createTextInput("JSONPath expression (e.g., $.name)", 40),
		inputsInitialized:  true,
	}
}

func createTextInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Width = width
	return ti
}

func createConnectionInputs() []textinput.Model {
	inputs := make([]textinput.Model, 5)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Connection Name"
	inputs[0].Focus()
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Host"
	inputs[1].Width = 30
	inputs[1].SetValue("localhost")

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Port"
	inputs[2].Width = 30
	inputs[2].SetValue("6379")

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Password (optional)"
	inputs[3].Width = 30
	inputs[3].EchoMode = textinput.EchoPassword

	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Database (0-15)"
	inputs[4].Width = 30
	inputs[4].SetValue("0")

	return inputs
}

func createAddKeyInputs() []textinput.Model {
	inputs := make([]textinput.Model, 3)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Key Name"
	inputs[0].Focus()
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Value"
	inputs[1].Width = 30

	// Third input: Field name (hash/stream) or Score (zset)
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Field"
	inputs[2].Width = 30

	return inputs
}

func createAddCollectionInputs() []textinput.Model {
	inputs := make([]textinput.Model, 2)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Field/Member"
	inputs[0].Focus()
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Value/Score"
	inputs[1].Width = 30

	return inputs
}

func createPubSubInputs() []textinput.Model {
	inputs := make([]textinput.Model, 2)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Channel"
	inputs[0].Focus()
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Message"
	inputs[1].Width = 30

	return inputs
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		cmd.LoadConnectionsCmd(),
		func() tea.Msg { return tea.EnableBracketedPaste() },
	}
	if m.CLIConnection != nil {
		conn := *m.CLIConnection
		cmds = append(cmds, func() tea.Msg {
			return types.AutoConnectMsg{Connection: conn}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) getPort() int {
	port, err := strconv.Atoi(m.ConnInputs[2].Value())
	if err != nil {
		return 6379
	}
	return port
}

func (m Model) getDB() int {
	db, err := strconv.Atoi(m.ConnInputs[4].Value())
	if err != nil {
		return 0
	}
	return db
}

func (m *Model) resetConnInputs() {
	for i := range m.ConnInputs {
		m.ConnInputs[i].SetValue("")
		m.ConnInputs[i].Blur()
	}
	m.ConnInputs[1].SetValue("localhost")
	m.ConnInputs[2].SetValue("6379")
	m.ConnInputs[4].SetValue("0")
	m.ConnInputs[0].Focus()
	m.ConnFocusIdx = 0
	m.ConnClusterMode = false
}

func (m *Model) resetAddKeyInputs() {
	for i := range m.AddKeyInputs {
		m.AddKeyInputs[i].SetValue("")
		m.AddKeyInputs[i].Blur()
	}
	if len(m.AddKeyInputs) > 0 {
		m.AddKeyInputs[0].Focus()
	}
	m.AddKeyFocusIdx = 0
	m.AddKeyType = types.KeyTypeString
}

func (m *Model) populateConnInputs(conn types.Connection) {
	m.ConnInputs[0].SetValue(conn.Name)
	m.ConnInputs[1].SetValue(conn.Host)
	m.ConnInputs[2].SetValue(strconv.Itoa(conn.Port))
	m.ConnInputs[3].SetValue(conn.Password)
	m.ConnInputs[4].SetValue(strconv.Itoa(conn.DB))
	m.ConnClusterMode = conn.UseCluster
}

// connFieldCount returns the number of focusable fields in the connection form.
// When cluster mode is on, the DB field is skipped (5 fields: name, host, port, password, cluster toggle).
// Otherwise there are 6 fields: name, host, port, password, cluster toggle, database.
func (m Model) connFieldCount() int {
	if m.ConnClusterMode {
		return 5
	}
	return 6
}

func (m *Model) resetAddCollectionInputs() {
	for i := range m.AddCollectionInput {
		m.AddCollectionInput[i].SetValue("")
		m.AddCollectionInput[i].Blur()
	}
	m.AddCollectionInput[0].Focus()
	m.AddCollFocusIdx = 0
}

func (m *Model) resetPubSubInputs() {
	for i := range m.PubSubInput {
		m.PubSubInput[i].SetValue("")
		m.PubSubInput[i].Blur()
	}
	m.PubSubInput[0].Focus()
	m.PubSubFocusIdx = 0
}
