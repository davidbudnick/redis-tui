package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestHandleConnectionsLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Loading = true
		msg := types.ConnectionsLoadedMsg{Connections: []types.Connection{{ID: 1, Name: "a"}}}
		result, cmd := m.handleConnectionsLoadedMsg(msg)
		model := result.(Model)
		if model.Loading {
			t.Error("expected Loading=false")
		}
		if len(model.Connections) != 1 {
			t.Errorf("expected 1 connection, got %d", len(model.Connections))
		}
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConnectionsLoadedMsg{Err: errors.New("boom")}
		result, _ := m.handleConnectionsLoadedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
		if model.Err == nil {
			t.Error("expected Err to be set")
		}
	})
}

func TestHandleConnectionAddedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenAddConnection
		msg := types.ConnectionAddedMsg{Connection: types.Connection{ID: 1, Name: "a"}}
		result, _ := m.handleConnectionAddedMsg(msg)
		model := result.(Model)
		if len(model.Connections) != 1 {
			t.Errorf("expected 1 connection, got %d", len(model.Connections))
		}
		if model.Screen != types.ScreenConnections {
			t.Errorf("expected ScreenConnections, got %v", model.Screen)
		}
		if model.StatusMsg != "Connection added" {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConnectionAddedMsg{Err: errors.New("boom")}
		result, _ := m.handleConnectionAddedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleConnectionUpdatedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = []types.Connection{{ID: 1, Name: "old"}, {ID: 2, Name: "other"}}
		m.EditingConnection = &types.Connection{ID: 1}
		m.Screen = types.ScreenEditConnection
		msg := types.ConnectionUpdatedMsg{Connection: types.Connection{ID: 1, Name: "new"}}
		result, _ := m.handleConnectionUpdatedMsg(msg)
		model := result.(Model)
		if model.Connections[0].Name != "new" {
			t.Errorf("expected updated name, got %q", model.Connections[0].Name)
		}
		if model.EditingConnection != nil {
			t.Error("expected EditingConnection cleared")
		}
		if model.Screen != types.ScreenConnections {
			t.Errorf("expected ScreenConnections, got %v", model.Screen)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConnectionUpdatedMsg{Err: errors.New("boom")}
		result, _ := m.handleConnectionUpdatedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleConnectionDeletedMsg(t *testing.T) {
	t.Run("success removes connection", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = []types.Connection{{ID: 1}, {ID: 2}, {ID: 3}}
		m.SelectedConnIdx = 2
		msg := types.ConnectionDeletedMsg{ID: 3}
		result, _ := m.handleConnectionDeletedMsg(msg)
		model := result.(Model)
		if len(model.Connections) != 2 {
			t.Errorf("expected 2 connections, got %d", len(model.Connections))
		}
		if model.SelectedConnIdx != 1 {
			t.Errorf("expected SelectedConnIdx=1, got %d", model.SelectedConnIdx)
		}
		if model.Screen != types.ScreenConnections {
			t.Errorf("expected ScreenConnections, got %v", model.Screen)
		}
	})
	t.Run("error preserves connections", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = []types.Connection{{ID: 1}}
		msg := types.ConnectionDeletedMsg{ID: 1, Err: errors.New("boom")}
		result, _ := m.handleConnectionDeletedMsg(msg)
		model := result.(Model)
		if len(model.Connections) != 1 {
			t.Errorf("expected 1 connection retained, got %d", len(model.Connections))
		}
	})
}

func TestHandleConnectedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{ID: 1}
		msg := types.ConnectedMsg{}
		result, cmd := m.handleConnectedMsg(msg)
		model := result.(Model)
		if model.Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", model.Screen)
		}
		if model.StatusMsg != "Connected" {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
		if cmd == nil {
			t.Error("expected non-nil cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConnectedMsg{Err: errors.New("boom")}
		result, cmd := m.handleConnectedMsg(msg)
		model := result.(Model)
		if model.ConnectionError == "" {
			t.Error("expected ConnectionError set")
		}
		if cmd != nil {
			t.Error("expected nil cmd on error")
		}
	})
	t.Run("cluster branch", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{ID: 1, UseCluster: true}
		msg := types.ConnectedMsg{}
		_, cmd := m.handleConnectedMsg(msg)
		if cmd == nil {
			t.Error("expected non-nil cmd for cluster branch")
		}
	})
	t.Run("with SendFunc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{ID: 1}
		fn := func(tea.Msg) {}
		m.SendFunc = &fn
		msg := types.ConnectedMsg{}
		_, cmd := m.handleConnectedMsg(msg)
		if cmd == nil {
			t.Error("expected non-nil cmd")
		}
	})
}

func TestHandleDisconnectedMsg(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.CurrentConn = &types.Connection{ID: 1}
	m.Keys = []types.RedisKey{{Key: "a"}}
	m.LiveMetrics = &types.LiveMetrics{}
	m.LiveMetricsActive = true
	result, cmd := m.handleDisconnectedMsg()
	model := result.(Model)
	if model.CurrentConn != nil {
		t.Error("expected CurrentConn cleared")
	}
	if model.Screen != types.ScreenConnections {
		t.Errorf("expected ScreenConnections, got %v", model.Screen)
	}
	if model.LiveMetricsActive {
		t.Error("expected LiveMetricsActive=false")
	}
	if cmd == nil {
		t.Error("expected non-nil unsubscribe cmd")
	}
}

func TestHandleConnectionTestMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConnectionTestMsg{Success: true, Latency: 2 * time.Millisecond}
		result, _ := m.handleConnectionTestMsg(msg)
		model := result.(Model)
		if !strings.Contains(model.TestConnResult, "Connected") {
			t.Errorf("unexpected result: %q", model.TestConnResult)
		}
	})
	t.Run("failure", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ConnectionTestMsg{Err: errors.New("boom")}
		result, _ := m.handleConnectionTestMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.TestConnResult, "Failed") {
			t.Errorf("unexpected result: %q", model.TestConnResult)
		}
	})
}

func TestHandleGroupsLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.GroupsLoadedMsg{Groups: []types.ConnectionGroup{{Name: "prod"}}}
		result, _ := m.handleGroupsLoadedMsg(msg)
		model := result.(Model)
		if len(model.ConnectionGroups) != 1 {
			t.Errorf("expected 1 group, got %d", len(model.ConnectionGroups))
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.GroupsLoadedMsg{Err: errors.New("boom")}
		result, _ := m.handleGroupsLoadedMsg(msg)
		model := result.(Model)
		if len(model.ConnectionGroups) != 0 {
			t.Errorf("expected 0 groups, got %d", len(model.ConnectionGroups))
		}
	})
}
