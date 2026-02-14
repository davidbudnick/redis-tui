package ui

import (
	"log/slog"

	"github.com/davidbudnick/redis-tui/internal/cmd"
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleAutoConnectMsg(msg types.AutoConnectMsg) (tea.Model, tea.Cmd) {
	conn := msg.Connection
	m.CurrentConn = &conn
	m.Loading = true
	m.StatusMsg = "Connecting..."
	m.CLIConnection = nil // Consume so it doesn't re-trigger
	return m, cmd.AutoConnectCmd(conn)
}

func (m Model) handleConnectionsLoadedMsg(msg types.ConnectionsLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		slog.Error("Failed to load connections", "error", msg.Err)
		m.Err = msg.Err
		m.StatusMsg = "Error: " + msg.Err.Error()
	} else {
		m.Connections = msg.Connections
		m.StatusMsg = ""
	}
	return m, nil
}

func (m Model) handleConnectionAddedMsg(msg types.ConnectionAddedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		slog.Error("Failed to add connection", "error", msg.Err)
		m.StatusMsg = "Error: " + msg.Err.Error()
	} else {
		m.Connections = append(m.Connections, msg.Connection)
		m.Screen = types.ScreenConnections
		m.resetConnInputs()
		m.StatusMsg = "Connection added"
	}
	return m, nil
}

func (m Model) handleConnectionUpdatedMsg(msg types.ConnectionUpdatedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
	} else {
		for i, c := range m.Connections {
			if c.ID == msg.Connection.ID {
				m.Connections[i] = msg.Connection
				break
			}
		}
		m.Screen = types.ScreenConnections
		m.EditingConnection = nil
		m.resetConnInputs()
		m.StatusMsg = "Connection updated"
	}
	return m, nil
}

func (m Model) handleConnectionDeletedMsg(msg types.ConnectionDeletedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		for i, c := range m.Connections {
			if c.ID == msg.ID {
				m.Connections = append(m.Connections[:i], m.Connections[i+1:]...)
				break
			}
		}
		if m.SelectedConnIdx >= len(m.Connections) && m.SelectedConnIdx > 0 {
			m.SelectedConnIdx--
		}
		m.StatusMsg = "Connection deleted"
	}
	m.Screen = types.ScreenConnections
	return m, nil
}

func (m Model) handleConnectedMsg(msg types.ConnectedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		slog.Error("Failed to connect", "error", msg.Err)
		m.ConnectionError = msg.Err.Error()
		m.StatusMsg = "Connection failed"
		return m, nil
	}
	m.ConnectionError = ""
	m.Keys = nil
	m.KeyCursor = 0
	m.SelectedKeyIdx = 0
	m.KeyPattern = ""
	m.CurrentKey = nil
	m.LiveMetrics = nil
	m.LiveMetricsActive = false
	m.Screen = types.ScreenKeys
	m.StatusMsg = "Connected"
	var sendFunc func(tea.Msg)
	if m.SendFunc != nil {
		sendFunc = *m.SendFunc
	}
	cmds := []tea.Cmd{cmd.LoadKeysCmd(m.KeyPattern, 0, 1000), cmd.SubscribeKeyspaceCmd("*", sendFunc), tickCmd()}
	if m.CurrentConn != nil && m.CurrentConn.UseCluster {
		cmds = append(cmds, cmd.FetchClusterNodesCmd())
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleDisconnectedMsg() (tea.Model, tea.Cmd) {
	m.CurrentConn = nil
	m.Keys = nil
	m.CurrentKey = nil
	m.LiveMetrics = nil
	m.LiveMetricsActive = false
	m.Screen = types.ScreenConnections
	m.StatusMsg = "Disconnected"
	return m, cmd.UnsubscribeKeyspaceCmd()
}

func (m Model) handleConnectionTestMsg(msg types.ConnectionTestMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.TestConnResult = "Failed: " + msg.Err.Error()
	} else {
		m.TestConnResult = "Connected in " + msg.Latency.String()
	}
	return m, nil
}

func (m Model) handleGroupsLoadedMsg(msg types.GroupsLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.ConnectionGroups = msg.Groups
	}
	return m, nil
}
