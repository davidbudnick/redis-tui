package cmd

import (
	"fmt"
	"log/slog"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

func (c *Commands) LoadConnections() tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionsLoadedMsg{Err: nil}
		}
		connections, err := c.config.ListConnections()
		if err != nil {
			slog.Error("Failed to load connections", "error", err)
		}
		return types.ConnectionsLoadedMsg{Connections: connections, Err: err}
	}
}

func (c *Commands) AddConnection(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionAddedMsg{Err: nil}
		}
		conn, err := c.config.AddConnection(conn)
		if err != nil {
			slog.Error("Failed to add connection", "error", err)
		}
		return types.ConnectionAddedMsg{Connection: conn, Err: err}
	}
}

func (c *Commands) UpdateConnection(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionUpdatedMsg{Err: nil}
		}
		conn, err := c.config.UpdateConnection(conn)
		if err != nil {
			slog.Error("Failed to update connection", "error", err)
		}
		return types.ConnectionUpdatedMsg{Connection: conn, Err: err}
	}
}

func (c *Commands) DeleteConnection(id int64) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionDeletedMsg{Err: nil}
		}
		err := c.config.DeleteConnection(id)
		return types.ConnectionDeletedMsg{ID: id, Err: err}
	}
}

func (c *Commands) Connect(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConnectedMsg{Err: nil}
		}
		var err error
		if conn.UseCluster {
			err = c.redis.ConnectCluster([]string{fmt.Sprintf("%s:%d", conn.Host, conn.Port)}, conn)
		} else {
			err = c.redis.Connect(conn)
		}
		if err != nil {
			slog.Error("Failed to connect", "error", err)
		}
		return types.ConnectedMsg{Err: err}
	}
}

func (c *Commands) Disconnect() tea.Cmd {
	return func() tea.Msg {
		if c.redis != nil {
			_ = c.redis.Disconnect()
		}
		return types.DisconnectedMsg{}
	}
}

func (c *Commands) TestConnection(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConnectionTestMsg{Success: false, Err: nil}
		}

		latency, err := c.redis.TestConnection(conn)
		return types.ConnectionTestMsg{Success: err == nil, Latency: latency, Err: err}
	}
}
