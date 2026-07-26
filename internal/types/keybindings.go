package types

// KeyBinding represents a single key binding
type KeyBinding struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// KeyBindings holds all customizable key bindings
type KeyBindings struct {
	// Navigation
	Up       string `json:"up"`
	Down     string `json:"down"`
	Left     string `json:"left"`
	Right    string `json:"right"`
	PageUp   string `json:"page_up"`
	PageDown string `json:"page_down"`
	Top      string `json:"top"`
	Bottom   string `json:"bottom"`

	// Actions
	Select  string `json:"select"`
	Back    string `json:"back"`
	Quit    string `json:"quit"`
	Help    string `json:"help"`
	Refresh string `json:"refresh"`
	Delete  string `json:"delete"`
	Add     string `json:"add"`
	Edit    string `json:"edit"`
	Copy    string `json:"copy"`
	Rename  string `json:"rename"`
	Search  string `json:"search"`
	Filter  string `json:"filter"`

	// Features
	Favorite      string `json:"favorite"`
	Watch         string `json:"watch"`
	Export        string `json:"export"`
	Import        string `json:"import"`
	ServerInfo    string `json:"server_info"`
	SlowLog       string `json:"slow_log"`
	LuaScript     string `json:"lua_script"`
	PubSub        string `json:"pubsub"`
	SwitchDB      string `json:"switch_db"`
	TTL           string `json:"ttl"`
	BulkDelete    string `json:"bulk_delete"`
	TreeView      string `json:"tree_view"`
	MemoryStats   string `json:"memory_stats"`
	ClientList    string `json:"client_list"`
	ClusterInfo   string `json:"cluster_info"`
	CompareKeys   string `json:"compare_keys"`
	JSONPath      string `json:"json_path"`
	CopyClipboard string `json:"copy_clipboard"`
	Logs          string `json:"logs"`
	Themes        string `json:"themes"`
	RecentKeys    string `json:"recent_keys"`
	Favorites     string `json:"favorites"`
	ValueHistory  string `json:"value_history"`
}

// DefaultKeyBindings returns the default key bindings
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		// Navigation
		Up:       "k",
		Down:     "j",
		Left:     "h",
		Right:    "l",
		PageUp:   "ctrl+u",
		PageDown: "ctrl+d",
		Top:      "g",
		Bottom:   "G",

		// Actions
		Select:  "enter",
		Back:    "esc",
		Quit:    "q",
		Help:    "?",
		Refresh: "r",
		Delete:  "d",
		Add:     "a",
		Edit:    "e",
		Copy:    "c",
		Rename:  "R",
		Search:  "/",
		Filter:  "f",

		// Features
		Favorite:      "F",
		Watch:         "w",
		Export:        "E",
		Import:        "I",
		ServerInfo:    "i",
		SlowLog:       "L",
		LuaScript:     "X",
		PubSub:        "p",
		SwitchDB:      "D",
		TTL:           "t",
		BulkDelete:    "B",
		TreeView:      "T",
		MemoryStats:   "M",
		ClientList:    "C",
		ClusterInfo:   "K",
		CompareKeys:   "=",
		JSONPath:      "J",
		CopyClipboard: "y",
		Logs:          "O",
		Themes:        "ctrl+t",
		RecentKeys:    "H",
		Favorites:     "ctrl+f",
		ValueHistory:  "u",
	}
}

// GetBindingsList returns key bindings as a list for display
func (kb KeyBindings) GetBindingsList() []KeyBinding {
	return []KeyBinding{
		{Key: kb.Up, Description: "Move up", Action: "up"},
		{Key: kb.Down, Description: "Move down", Action: "down"},
		{Key: kb.PageUp, Description: "Page up", Action: "page_up"},
		{Key: kb.PageDown, Description: "Page down", Action: "page_down"},
		{Key: kb.Top, Description: "Go to top", Action: "top"},
		{Key: kb.Bottom, Description: "Go to bottom", Action: "bottom"},
		{Key: kb.Select, Description: "Select/Enter", Action: "select"},
		{Key: kb.Back, Description: "Go back", Action: "back"},
		{Key: kb.Quit, Description: "Quit", Action: "quit"},
		{Key: kb.Help, Description: "Show help", Action: "help"},
		{Key: kb.Refresh, Description: "Refresh", Action: "refresh"},
		{Key: kb.Delete, Description: "Delete", Action: "delete"},
		{Key: kb.Add, Description: "Add new", Action: "add"},
		{Key: kb.Edit, Description: "Edit", Action: "edit"},
		{Key: kb.Copy, Description: "Copy key", Action: "copy"},
		{Key: kb.Rename, Description: "Rename", Action: "rename"},
		{Key: kb.Search, Description: "Search", Action: "search"},
		{Key: kb.Filter, Description: "Filter", Action: "filter"},
		{Key: kb.Favorite, Description: "Toggle favorite", Action: "favorite"},
		{Key: kb.Watch, Description: "Watch key", Action: "watch"},
		{Key: kb.Export, Description: "Export keys", Action: "export"},
		{Key: kb.Import, Description: "Import keys", Action: "import"},
		{Key: kb.ServerInfo, Description: "Server info", Action: "server_info"},
		{Key: kb.SlowLog, Description: "Slow log", Action: "slow_log"},
		{Key: kb.LuaScript, Description: "Lua script", Action: "lua_script"},
		{Key: kb.PubSub, Description: "Pub/Sub", Action: "pubsub"},
		{Key: kb.SwitchDB, Description: "Switch database", Action: "switch_db"},
		{Key: kb.TTL, Description: "Set TTL", Action: "ttl"},
		{Key: kb.BulkDelete, Description: "Bulk delete", Action: "bulk_delete"},
		{Key: kb.TreeView, Description: "Tree view", Action: "tree_view"},
		{Key: kb.MemoryStats, Description: "Memory stats", Action: "memory_stats"},
		{Key: kb.ClientList, Description: "Client list", Action: "client_list"},
		{Key: kb.ClusterInfo, Description: "Cluster info", Action: "cluster_info"},
		{Key: kb.CompareKeys, Description: "Compare keys", Action: "compare_keys"},
		{Key: kb.JSONPath, Description: "JSON path query", Action: "json_path"},
		{Key: kb.CopyClipboard, Description: "Copy to clipboard", Action: "copy_clipboard"},
		{Key: kb.Logs, Description: "View logs", Action: "logs"},
		{Key: kb.RecentKeys, Description: "Recent keys", Action: "recent_keys"},
		{Key: kb.Favorites, Description: "Favorites", Action: "favorites"},
		{Key: kb.ValueHistory, Description: "Value history", Action: "value_history"},
	}
}
