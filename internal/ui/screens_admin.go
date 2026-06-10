package ui

import (
	"strconv"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleHelpScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "?":
		if m.CurrentConn != nil {
			m.Screen = types.ScreenKeys
		} else {
			m.Screen = types.ScreenConnections
		}
	}
	return m, nil
}

func (m Model) handleServerInfoScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Screen = types.ScreenKeys
	case "r":
		m.Loading = true
		return m, m.Cmds.LoadServerInfo()
	}
	return m, nil
}

func (m Model) handlePubSubScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.PubSubInput[m.PubSubFocusIdx].Blur()
		m.PubSubFocusIdx = (m.PubSubFocusIdx + 1) % len(m.PubSubInput)
		m.PubSubInput[m.PubSubFocusIdx].Focus()
	case "enter":
		if m.PubSubInput[0].Value() != "" && m.PubSubInput[1].Value() != "" {
			m.Loading = true
			return m, m.Cmds.PublishMessage(m.PubSubInput[0].Value(), m.PubSubInput[1].Value())
		}
	case "esc":
		m.Screen = types.ScreenPubSubChannels
		m.resetPubSubInputs()
	default:
		var cmds []tea.Cmd
		for i := range m.PubSubInput {
			var inputCmd tea.Cmd
			m.PubSubInput[i], inputCmd = m.PubSubInput[i].Update(msg)
			cmds = append(cmds, inputCmd)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) handlePublishMessageScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handlePubSubScreen(msg)
}

func (m Model) handleSwitchDBScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		dbNum, err := strconv.Atoi(m.Inputs.DBSwitchInput.Value())
		if err == nil && dbNum >= 0 && dbNum <= 15 {
			m.Loading = true
			return m, m.Cmds.SwitchDB(dbNum)
		} else {
			m.StatusMsg = "Invalid database number (0-15)"
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.DBSwitchInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.DBSwitchInput, inputCmd = m.Inputs.DBSwitchInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleExportScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Inputs.ExportInput.Value() != "" {
			m.Loading = true
			pattern := m.KeyPattern
			if pattern == "" {
				pattern = "*"
			}
			return m, m.Cmds.ExportKeys(pattern, m.Inputs.ExportInput.Value())
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.ExportInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.ExportInput, inputCmd = m.Inputs.ExportInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleImportScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Inputs.ImportInput.Value() != "" {
			m.Loading = true
			return m, m.Cmds.ImportKeys(m.Inputs.ImportInput.Value())
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.ImportInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.ImportInput, inputCmd = m.Inputs.ImportInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleSlowLogScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Screen = types.ScreenKeys
	case "r":
		m.Loading = true
		return m, m.Cmds.GetSlowLog(20)
	}
	return m, nil
}

func (m Model) handleLuaScriptScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Inputs.LuaScriptInput.Value() != "" {
			m.Loading = true
			return m, m.Cmds.EvalLuaScript(m.Inputs.LuaScriptInput.Value(), []string{})
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.LuaScriptInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.LuaScriptInput, inputCmd = m.Inputs.LuaScriptInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleLogsScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.ShowingLogDetail {
		switch msg.String() {
		case "esc", "enter":
			m.ShowingLogDetail = false
		}
		return m, nil
	}

	logCount := 0
	if m.Logs != nil {
		logCount = m.Logs.Len()
	}

	switch msg.String() {
	case "esc":
		m.Screen = types.ScreenKeys
	case "up", "k":
		if m.LogCursor > 0 {
			m.LogCursor--
		}
	case "down", "j":
		if m.LogCursor < logCount-1 {
			m.LogCursor++
		}
	case "enter":
		if logCount > 0 {
			m.ShowingLogDetail = true
		}
	case "g":
		m.LogCursor = 0
	case "G":
		if logCount > 0 {
			m.LogCursor = logCount - 1
		}
	}
	return m, nil
}

func (m Model) handleClientListScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedClientIdx > 0 {
			m.SelectedClientIdx--
		}
	case "down", "j":
		if m.SelectedClientIdx < len(m.ClientList)-1 {
			m.SelectedClientIdx++
		}
	case "r":
		m.Loading = true
		return m, m.Cmds.GetClientList()
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleMemoryStatsScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.Loading = true
		return m, m.Cmds.GetMemoryStats()
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleClusterInfoScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedNodeIdx > 0 {
			m.SelectedNodeIdx--
		}
	case "down", "j":
		if m.SelectedNodeIdx < len(m.ClusterNodes)-1 {
			m.SelectedNodeIdx++
		}
	case "r":
		m.Loading = true
		return m, m.Cmds.GetClusterInfo()
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handlePubSubChannelsScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedChannelIdx > 0 {
			m.SelectedChannelIdx--
		}
	case "down", "j":
		if m.SelectedChannelIdx < len(m.PubSubChannels)-1 {
			m.SelectedChannelIdx++
		}
	case "r":
		m.Loading = true
		return m, m.Cmds.GetPubSubChannels("*")
	case "p":
		m.Screen = types.ScreenPubSub
		m.resetPubSubInputs()
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleRedisConfigScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.EditingConfigParam != "" {
		switch msg.String() {
		case "enter":
			m.Loading = true
			param := m.EditingConfigParam
			value := m.Inputs.ConfigEditInput.Value()
			m.EditingConfigParam = ""
			m.Inputs.ConfigEditInput.Blur()
			return m, m.Cmds.SetRedisConfig(param, value)
		case "esc":
			m.EditingConfigParam = ""
			m.Inputs.ConfigEditInput.Blur()
		default:
			var inputCmd tea.Cmd
			m.Inputs.ConfigEditInput, inputCmd = m.Inputs.ConfigEditInput.Update(msg)
			return m, inputCmd
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.SelectedConfigIdx > 0 {
			m.SelectedConfigIdx--
		}
	case "down", "j":
		if m.SelectedConfigIdx < len(m.RedisConfigParams)-1 {
			m.SelectedConfigIdx++
		}
	case "e", "enter":
		if len(m.RedisConfigParams) > 0 && m.SelectedConfigIdx < len(m.RedisConfigParams) {
			param := m.RedisConfigParams[m.SelectedConfigIdx]
			m.EditingConfigParam = param.Name
			m.Inputs.ConfigEditInput.SetValue(param.Value)
			m.Inputs.ConfigEditInput.Focus()
		}
	case "r":
		m.Loading = true
		return m, m.Cmds.LoadRedisConfig("*")
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleLiveMetricsScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "c":
		if m.LiveMetrics != nil {
			m.LiveMetrics.DataPoints = nil
		}
	case "q", "esc":
		m.LiveMetricsActive = false
		m.Screen = types.ScreenKeys
	}
	return m, nil
}
