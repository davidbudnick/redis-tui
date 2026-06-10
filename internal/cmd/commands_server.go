package cmd

import (
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

func (c *Commands) LoadServerInfo() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ServerInfoLoadedMsg{Err: nil}
		}
		info, err := c.redis.GetServerInfo()
		return types.ServerInfoLoadedMsg{Info: info, Err: err}
	}
}

func (c *Commands) FlushDB() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.FlushDBMsg{Err: nil}
		}
		err := c.redis.FlushDB()
		return types.FlushDBMsg{Err: err}
	}
}

func (c *Commands) GetMemoryUsage(key string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.MemoryUsageMsg{Key: key, Err: nil}
		}
		bytes, err := c.redis.MemoryUsage(key)
		return types.MemoryUsageMsg{Key: key, Bytes: bytes, Err: err}
	}
}

func (c *Commands) GetSlowLog(count int64) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.SlowLogLoadedMsg{Err: nil}
		}
		entries, err := c.redis.SlowLogGet(count)
		return types.SlowLogLoadedMsg{Entries: entries, Err: err}
	}
}

func (c *Commands) GetClientList() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ClientListLoadedMsg{Err: nil}
		}
		clients, err := c.redis.ClientList()
		return types.ClientListLoadedMsg{Clients: clients, Err: err}
	}
}

func (c *Commands) GetMemoryStats() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.MemoryStatsLoadedMsg{Err: nil}
		}
		stats, err := c.redis.GetMemoryStats()
		return types.MemoryStatsLoadedMsg{Stats: stats, Err: err}
	}
}

func (c *Commands) GetClusterInfo() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ClusterInfoLoadedMsg{Err: nil}
		}
		nodes, err := c.redis.ClusterNodes()
		info, _ := c.redis.ClusterInfo()
		return types.ClusterInfoLoadedMsg{Nodes: nodes, Info: info, Err: err}
	}
}

func (c *Commands) FetchClusterNodes() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ClusterNodesLoadedMsg{Err: nil}
		}
		nodes, err := c.redis.ClusterNodes()
		return types.ClusterNodesLoadedMsg{Nodes: nodes, Err: err}
	}
}

func (c *Commands) LoadLiveMetrics() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.LiveMetricsMsg{Err: nil}
		}
		data, err := c.redis.GetLiveMetrics()
		return types.LiveMetricsMsg{Data: data, Err: err}
	}
}
