package ui

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleServerInfoLoadedMsg(msg types.ServerInfoLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.ServerInfo = msg.Info
		m.Screen = types.ScreenServerInfo
	}
	return m, nil
}

func (m Model) handleDBSwitchedMsg(msg types.DBSwitchedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	if m.CurrentConn != nil {
		m.CurrentConn.DB = msg.DB
	}
	m.StatusMsg = "Switched to database " + strconv.Itoa(msg.DB)
	m.Screen = types.ScreenKeys
	m.KeyCursor = 0
	m.Keys = []types.RedisKey{}
	return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
}

func (m Model) handleFlushDBMsg(msg types.FlushDBMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.Keys = []types.RedisKey{}
		m.StatusMsg = "Database flushed"
	}
	m.Screen = types.ScreenKeys
	return m, nil
}

func (m Model) handleSlowLogLoadedMsg(msg types.SlowLogLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
	} else {
		m.SlowLogEntries = msg.Entries
		m.Screen = types.ScreenSlowLog
	}
	return m, nil
}

func (m Model) handleClientListLoadedMsg(msg types.ClientListLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.ClientList = msg.Clients
		m.Screen = types.ScreenClientList
	}
	return m, nil
}

func (m Model) handleMemoryStatsLoadedMsg(msg types.MemoryStatsLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.MemoryStats = &msg.Stats
		m.Screen = types.ScreenMemoryStats
	}
	return m, nil
}

func (m Model) handleClusterInfoLoadedMsg(msg types.ClusterInfoLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.ClusterNodes = msg.Nodes
		m.ClusterEnabled = len(msg.Nodes) > 0
		m.Screen = types.ScreenClusterInfo
	}
	return m, nil
}

func (m Model) handleClusterNodesLoadedMsg(msg types.ClusterNodesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		m.ClusterNodes = msg.Nodes
		m.ClusterEnabled = len(msg.Nodes) > 0
	}
	return m, nil
}

func (m Model) handleMemoryUsageMsg(msg types.MemoryUsageMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.MemoryUsage = msg.Bytes
	}
	return m, nil
}

func (m Model) handlePubSubChannelsLoadedMsg(msg types.PubSubChannelsLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	m.PubSubChannels = msg.Channels
	m.SelectedChannelIdx = 0
	m.Screen = types.ScreenPubSubChannels
	return m, nil
}

func (m Model) handleConfigLoadedMsg(msg types.ConfigLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	params := make([]types.RedisConfigParam, 0, len(msg.Params))
	for name, value := range msg.Params {
		params = append(params, types.RedisConfigParam{Name: name, Value: value})
	}
	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})
	m.RedisConfigParams = params
	m.SelectedConfigIdx = 0
	m.Screen = types.ScreenRedisConfig
	return m, nil
}

func (m Model) handleConfigSetMsg(msg types.ConfigSetMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = fmt.Sprintf("Config updated: %s = %s", msg.Param, msg.Value)
	return m, m.Cmds.LoadRedisConfig("*")
}

// Script and Pub/Sub handlers

func (m Model) handleLuaScriptResultMsg(msg types.LuaScriptResultMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.LuaResult = "Error: " + msg.Err.Error()
	} else {
		switch v := msg.Result.(type) {
		case string:
			m.LuaResult = v
		case int64:
			m.LuaResult = strconv.FormatInt(v, 10)
		case []any:
			m.LuaResult = "Array result (length: " + strconv.Itoa(len(v)) + ")"
		default:
			m.LuaResult = "OK"
		}
	}
	return m, nil
}

func (m Model) handlePublishResultMsg(msg types.PublishResultMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Publish failed: " + msg.Err.Error()
	} else {
		m.StatusMsg = "Message sent to " + strconv.FormatInt(msg.Receivers, 10) + " subscribers"
		m.Screen = types.ScreenPubSubChannels
	}
	return m, nil
}

func (m Model) handleKeyspaceEventMsg(msg types.KeyspaceEventMsg) (tea.Model, tea.Cmd) {
	m.KeyspaceEvents = append(m.KeyspaceEvents, msg.Event)
	if len(m.KeyspaceEvents) > 100 {
		// Create new slice to allow GC of old backing array (prevents memory leak)
		newEvents := make([]types.KeyspaceEvent, 99)
		copy(newEvents, m.KeyspaceEvents[1:])
		m.KeyspaceEvents = newEvents
	}
	// Refresh keys if a key was set or deleted
	if msg.Event.Event == "set" || msg.Event.Event == "del" {
		m.StatusMsg = fmt.Sprintf("Key %s: %s", msg.Event.Event, msg.Event.Key)
		return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
	}
	return m, nil
}

func (m Model) handleLiveMetricsMsg(msg types.LiveMetricsMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	if m.LiveMetrics == nil {
		m.LiveMetrics = &types.LiveMetrics{
			MaxDataPoints:   60,
			RefreshInterval: time.Second,
		}
	}
	m.LiveMetrics.DataPoints = append(m.LiveMetrics.DataPoints, msg.Data)
	if len(m.LiveMetrics.DataPoints) > m.LiveMetrics.MaxDataPoints {
		newPoints := make([]types.LiveMetricsData, m.LiveMetrics.MaxDataPoints)
		copy(newPoints, m.LiveMetrics.DataPoints[1:])
		m.LiveMetrics.DataPoints = newPoints
	}
	m.LiveMetricsActive = true
	m.Screen = types.ScreenLiveMetrics
	return m, nil
}

func (m Model) handleLiveMetricsTickMsg() (tea.Model, tea.Cmd) {
	if !m.LiveMetricsActive {
		return m, nil
	}
	return m, m.Cmds.LoadLiveMetrics()
}
