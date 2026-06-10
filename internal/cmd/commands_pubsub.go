package cmd

import (
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

func (c *Commands) PublishMessage(channel, message string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.PublishResultMsg{Channel: channel, Err: nil}
		}
		receivers, err := c.redis.Publish(channel, message)
		return types.PublishResultMsg{Channel: channel, Receivers: receivers, Err: err}
	}
}

func (c *Commands) GetPubSubChannels(pattern string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.PubSubChannelsLoadedMsg{Err: nil}
		}
		names, err := c.redis.PubSubChannels(pattern)
		if err != nil {
			return types.PubSubChannelsLoadedMsg{Err: err}
		}
		channels := make([]types.PubSubChannel, len(names))
		for i, name := range names {
			channels[i] = types.PubSubChannel{Name: name}
		}
		return types.PubSubChannelsLoadedMsg{Channels: channels}
	}
}

func (c *Commands) SubscribeKeyspace(pattern string, sendFunc func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeyspaceSubscribedMsg{Err: nil}
		}
		err := c.redis.SubscribeKeyspace(pattern, func(event types.KeyspaceEvent) {
			if sendFunc != nil {
				sendFunc(types.KeyspaceEventMsg{Event: event})
			}
		})
		return types.KeyspaceSubscribedMsg{Pattern: pattern, Err: err}
	}
}

func (c *Commands) UnsubscribeKeyspace() tea.Cmd {
	return func() tea.Msg {
		if c.redis != nil {
			_ = c.redis.UnsubscribeKeyspace()
		}
		return nil
	}
}
