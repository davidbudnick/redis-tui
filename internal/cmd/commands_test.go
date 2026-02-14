package cmd

import (
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/service"
	"github.com/davidbudnick/redis-tui/internal/testutil"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// --- Config command tests (use real testutil.NewTestConfig) ---

func TestLoadConnections(t *testing.T) {
	t.Run("success empty", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		cmds := NewCommands(cfg, nil)
		msg := cmds.LoadConnections()()
		result, ok := msg.(types.ConnectionsLoadedMsg)
		if !ok {
			t.Fatalf("expected ConnectionsLoadedMsg, got %T", msg)
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Connections) != 0 {
			t.Errorf("expected 0 connections, got %d", len(result.Connections))
		}
	})

	t.Run("success with connections", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		testutil.MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
		cmds := NewCommands(cfg, nil)
		msg := cmds.LoadConnections()()
		result := msg.(types.ConnectionsLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Connections) != 1 {
			t.Errorf("expected 1 connection, got %d", len(result.Connections))
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadConnections()()
		result := msg.(types.ConnectionsLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestAddConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		cmds := NewCommands(cfg, nil)
		msg := cmds.AddConnection("test", "localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectionAddedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Connection.Name != "test" {
			t.Errorf("Name = %q, want %q", result.Connection.Name, "test")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.AddConnection("test", "localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectionAddedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestUpdateConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		conn := testutil.MustAddConnection(t, cfg, "old", "localhost", 6379, "", 0)
		cmds := NewCommands(cfg, nil)
		msg := cmds.UpdateConnection(conn.ID, "new", "localhost", 6380, "pass", 1, false)()
		result := msg.(types.ConnectionUpdatedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Connection.Name != "new" {
			t.Errorf("Name = %q, want %q", result.Connection.Name, "new")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.UpdateConnection(1, "n", "h", 1, "p", 0, false)()
		result := msg.(types.ConnectionUpdatedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestDeleteConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		conn := testutil.MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
		cmds := NewCommands(cfg, nil)
		msg := cmds.DeleteConnection(conn.ID)()
		result := msg.(types.ConnectionDeletedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.ID != conn.ID {
			t.Errorf("ID = %d, want %d", result.ID, conn.ID)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.DeleteConnection(1)()
		result := msg.(types.ConnectionDeletedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestLoadFavorites(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.LoadFavorites(1)()
	result := msg.(types.FavoritesLoadedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddFavorite(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	testutil.MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
	cmds := NewCommands(cfg, nil)
	msg := cmds.AddFavorite(1, "mykey", "My Key")()
	result := msg.(types.FavoriteAddedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFavorite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := testutil.NewTestConfig(t)
		testutil.MustAddConnection(t, cfg, "test", "localhost", 6379, "", 0)
		_, _ = cfg.AddFavorite(1, "mykey", "label")
		cmds := NewCommands(cfg, nil)
		msg := cmds.RemoveFavorite(1, "mykey")()
		result := msg.(types.FavoriteRemovedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.RemoveFavorite(1, "mykey")()
		result := msg.(types.FavoriteRemovedMsg)
		if result.Err != nil {
			t.Errorf("nil config should not error: %v", result.Err)
		}
	})
}

func TestLoadRecentKeys(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.LoadRecentKeys(1)()
	result := msg.(types.RecentKeysLoadedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddRecentKey(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.AddRecentKey(1, "mykey", types.KeyTypeString)()
	// AddRecentKey returns nil
	if msg != nil {
		t.Errorf("expected nil msg, got %T", msg)
	}
}

func TestLoadTemplates(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.LoadTemplates()()
	result := msg.(types.TemplatesLoadedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestLoadValueHistory(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.LoadValueHistory("mykey")()
	result := msg.(types.ValueHistoryMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestSaveValueHistory(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cmds := NewCommands(cfg, nil)
	msg := cmds.SaveValueHistory("mykey", types.RedisValue{StringValue: "val"}, "set")()
	if msg != nil {
		t.Errorf("expected nil msg, got %T", msg)
	}
}

// --- Redis command tests (use FullMockRedisClient) ---

func newMockCmds() (*Commands, *testutil.FullMockRedisClient) {
	mock := testutil.NewFullMockRedisClient()
	cmds := NewCommands(nil, mock)
	return cmds, mock
}

func TestConnect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.Connect("localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectError = errors.New("connection refused")
		msg := cmds.Connect("localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("cluster mode", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.Connect("localhost", 7000, "", 0, true)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.Connect("localhost", 6379, "", 0, false)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestDisconnect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.Disconnect()()
		if _, ok := msg.(types.DisconnectedMsg); !ok {
			t.Fatalf("expected DisconnectedMsg, got %T", msg)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.Disconnect()()
		if _, ok := msg.(types.DisconnectedMsg); !ok {
			t.Fatalf("expected DisconnectedMsg, got %T", msg)
		}
	})
}

func TestLoadKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		mock.SetKey("k1", types.RedisValue{}, types.KeyTypeString, 0)
		msg := cmds.LoadKeys("*", 0, 100)()
		result := msg.(types.KeysLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(result.Keys))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadKeys("*", 0, 100)()
		result := msg.(types.KeysLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestLoadKeyValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		mock.SetKey("mykey", types.RedisValue{Type: types.KeyTypeString, StringValue: "val"}, types.KeyTypeString, 0)
		msg := cmds.LoadKeyValue("mykey")()
		result := msg.(types.KeyValueLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key != "mykey" {
			t.Errorf("Key = %q, want %q", result.Key, "mykey")
		}
		if result.Value.StringValue != "val" {
			t.Errorf("StringValue = %q, want %q", result.Value.StringValue, "val")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadKeyValue("mykey")()
		result := msg.(types.KeyValueLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestLoadKeyPreview(t *testing.T) {
	cmds, mock := newMockCmds()
	_ = mock.Connect("localhost", 6379, "", 0)
	mock.SetKey("pk", types.RedisValue{Type: types.KeyTypeString, StringValue: "preview"}, types.KeyTypeString, 0)
	msg := cmds.LoadKeyPreview("pk")()
	result := msg.(types.KeyPreviewLoadedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
	if result.Key != "pk" {
		t.Errorf("Key = %q, want %q", result.Key, "pk")
	}
}

func TestDeleteKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		msg := cmds.DeleteKey("mykey")()
		result := msg.(types.KeyDeletedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key != "mykey" {
			t.Errorf("Key = %q, want %q", result.Key, "mykey")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.DeleteKey("mykey")()
		result := msg.(types.KeyDeletedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestSetTTL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.SetTTL("mykey", 60*time.Second)()
		result := msg.(types.TTLSetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.TTL != 60*time.Second {
			t.Errorf("TTL = %v, want %v", result.TTL, 60*time.Second)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SetTTLError = errors.New("ttl error")
		msg := cmds.SetTTL("mykey", time.Second)()
		result := msg.(types.TTLSetMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SetTTL("mykey", time.Second)()
		result := msg.(types.TTLSetMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestCreateKey(t *testing.T) {
	keyTypes := []struct {
		name    string
		keyType types.KeyType
		value   string
		extra   string
	}{
		{"string", types.KeyTypeString, "hello", ""},
		{"list", types.KeyTypeList, "item1", ""},
		{"set", types.KeyTypeSet, "member1", ""},
		{"zset", types.KeyTypeZSet, "member1", "1.5"},
		{"hash", types.KeyTypeHash, "value1", "field1"},
		{"stream", types.KeyTypeStream, "value1", "data"},
	}

	for _, tt := range keyTypes {
		t.Run(tt.name, func(t *testing.T) {
			cmds, mock := newMockCmds()
			_ = mock.Connect("localhost", 6379, "", 0)
			msg := cmds.CreateKey("newkey", tt.keyType, tt.value, tt.extra, 0)()
			result := msg.(types.KeySetMsg)
			if result.Err != nil {
				t.Errorf("unexpected error for %s: %v", tt.name, result.Err)
			}
			if result.Key != "newkey" {
				t.Errorf("Key = %q, want %q", result.Key, "newkey")
			}
		})
	}

	t.Run("zset with empty extra defaults to 0", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		msg := cmds.CreateKey("zkey", types.KeyTypeZSet, "member", "", 0)()
		result := msg.(types.KeySetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("hash with empty extra defaults to 'field'", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		msg := cmds.CreateKey("hkey", types.KeyTypeHash, "val", "", 0)()
		result := msg.(types.KeySetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("stream with empty extra defaults to 'data'", func(t *testing.T) {
		cmds, mock := newMockCmds()
		_ = mock.Connect("localhost", 6379, "", 0)
		msg := cmds.CreateKey("skey", types.KeyTypeStream, "val", "", 0)()
		result := msg.(types.KeySetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CreateKey("k", types.KeyTypeString, "v", "", 0)()
		result := msg.(types.KeySetMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestEditStringValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.EditStringValue("mykey", "newval")()
		result := msg.(types.ValueEditedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SetStringError = errors.New("set error")
		msg := cmds.EditStringValue("mykey", "val")()
		result := msg.(types.ValueEditedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.EditStringValue("k", "v")()
		result := msg.(types.ValueEditedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestEditListElement(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.EditListElement("list", 0, "newval")()
	result := msg.(types.ValueEditedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestEditHashField(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.EditHashField("hash", "field", "val")()
	result := msg.(types.ValueEditedMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddToList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.AddToList("list", "item1", "item2")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.AddToList("list", "item")()
		result := msg.(types.ItemAddedToCollectionMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestAddToSet(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.AddToSet("set", "member1", "member2")()
	result := msg.(types.ItemAddedToCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddToZSet(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.AddToZSet("zset", 1.5, "member")()
	result := msg.(types.ItemAddedToCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddToHash(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.AddToHash("hash", "field", "value")()
	result := msg.(types.ItemAddedToCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestAddToStream(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.AddToStream("stream", map[string]interface{}{"key": "val"})()
	result := msg.(types.ItemAddedToCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromList(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromList("list", "item")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromSet(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromSet("set", "member")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromZSet(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromZSet("zset", "member")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromHash(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromHash("hash", "field")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRemoveFromStream(t *testing.T) {
	cmds, _ := newMockCmds()
	msg := cmds.RemoveFromStream("stream", "1-0")()
	result := msg.(types.ItemRemovedFromCollectionMsg)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestRenameKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.RenameKey("old", "new")()
		result := msg.(types.KeyRenamedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.OldKey != "old" || result.NewKey != "new" {
			t.Errorf("OldKey=%q NewKey=%q, want old/new", result.OldKey, result.NewKey)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.RenameError = errors.New("rename error")
		msg := cmds.RenameKey("old", "new")()
		result := msg.(types.KeyRenamedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.RenameKey("old", "new")()
		result := msg.(types.KeyRenamedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestCopyKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.CopyKey("src", "dst", true)()
		result := msg.(types.KeyCopiedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.SourceKey != "src" || result.DestKey != "dst" {
			t.Errorf("got src=%q dst=%q", result.SourceKey, result.DestKey)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CopyKey("src", "dst", false)()
		result := msg.(types.KeyCopiedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestSwitchDB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.SwitchDB(1)()
		result := msg.(types.DBSwitchedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.DB != 1 {
			t.Errorf("DB = %d, want 1", result.DB)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SelectDBError = errors.New("select error")
		msg := cmds.SwitchDB(2)()
		result := msg.(types.DBSwitchedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SwitchDB(0)()
		result := msg.(types.DBSwitchedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestLoadServerInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ServerInfo = types.ServerInfo{Version: "7.0.0"}
		msg := cmds.LoadServerInfo()()
		result := msg.(types.ServerInfoLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Info.Version != "7.0.0" {
			t.Errorf("Version = %q, want %q", result.Info.Version, "7.0.0")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ServerInfoError = errors.New("info error")
		msg := cmds.LoadServerInfo()()
		result := msg.(types.ServerInfoLoadedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadServerInfo()()
		result := msg.(types.ServerInfoLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestFlushDB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		msg := cmds.FlushDB()()
		result := msg.(types.FlushDBMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.FlushDBError = errors.New("flush error")
		msg := cmds.FlushDB()()
		result := msg.(types.FlushDBMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.FlushDB()()
		result := msg.(types.FlushDBMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetMemoryUsage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.MemUsageResult = 1024
		msg := cmds.GetMemoryUsage("mykey")()
		result := msg.(types.MemoryUsageMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Bytes != 1024 {
			t.Errorf("Bytes = %d, want 1024", result.Bytes)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetMemoryUsage("k")()
		result := msg.(types.MemoryUsageMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetSlowLog(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SlowLogEntries = []types.SlowLogEntry{{ID: 1, Command: "GET key"}}
		msg := cmds.GetSlowLog(10)()
		result := msg.(types.SlowLogLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(result.Entries))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetSlowLog(10)()
		result := msg.(types.SlowLogLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetClientList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ClientListResult = []types.ClientInfo{{ID: 1, Addr: "127.0.0.1:1234"}}
		msg := cmds.GetClientList()()
		result := msg.(types.ClientListLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Clients) != 1 {
			t.Errorf("expected 1 client, got %d", len(result.Clients))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetClientList()()
		result := msg.(types.ClientListLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetMemoryStats(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.MemStats = types.MemoryStats{UsedMemory: 1024}
		msg := cmds.GetMemoryStats()()
		result := msg.(types.MemoryStatsLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Stats.UsedMemory != 1024 {
			t.Errorf("UsedMemory = %d, want 1024", result.Stats.UsedMemory)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetMemoryStats()()
		result := msg.(types.MemoryStatsLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetClusterInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ClusterNodesResult = []types.ClusterNode{{ID: "abc"}}
		mock.ClusterInfoResult = "cluster_state:ok"
		msg := cmds.GetClusterInfo()()
		result := msg.(types.ClusterInfoLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Nodes) != 1 {
			t.Errorf("expected 1 node, got %d", len(result.Nodes))
		}
		if result.Info != "cluster_state:ok" {
			t.Errorf("Info = %q, want %q", result.Info, "cluster_state:ok")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetClusterInfo()()
		result := msg.(types.ClusterInfoLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestSearchByValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SearchByValueResult = []types.RedisKey{{Key: "found"}}
		msg := cmds.SearchByValue("*", "needle", 100)()
		result := msg.(types.KeysLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(result.Keys))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SearchByValue("*", "v", 10)()
		result := msg.(types.KeysLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestRegexSearch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.RegexSearchResult = []types.RedisKey{{Key: "user:123"}}
		msg := cmds.RegexSearch("user:\\d+", 100)()
		result := msg.(types.RegexSearchResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(result.Keys))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.RegexSearch(".*", 10)()
		result := msg.(types.RegexSearchResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestFuzzySearch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.FuzzySearchResult = []types.RedisKey{{Key: "user:abc"}}
		msg := cmds.FuzzySearch("usr", 100)()
		result := msg.(types.FuzzySearchResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(result.Keys))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.FuzzySearch("test", 10)()
		result := msg.(types.FuzzySearchResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestCompareKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.CompareValue1 = types.RedisValue{StringValue: "val1"}
		mock.CompareValue2 = types.RedisValue{StringValue: "val2"}
		msg := cmds.CompareKeys("k1", "k2")()
		result := msg.(types.CompareKeysResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Key1Value.StringValue != "val1" {
			t.Errorf("Key1Value.StringValue = %q, want %q", result.Key1Value.StringValue, "val1")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.CompareKeys("k1", "k2")()
		result := msg.(types.CompareKeysResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestBulkDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.BulkDeleteResult = 5
		msg := cmds.BulkDelete("user:*")()
		result := msg.(types.BulkDeleteMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Deleted != 5 {
			t.Errorf("Deleted = %d, want 5", result.Deleted)
		}
		if result.Pattern != "user:*" {
			t.Errorf("Pattern = %q, want %q", result.Pattern, "user:*")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.BulkDeleteError = errors.New("bulk error")
		msg := cmds.BulkDelete("*")()
		result := msg.(types.BulkDeleteMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.BulkDelete("*")()
		result := msg.(types.BulkDeleteMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestBatchSetTTL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.BatchTTLResult = 3
		msg := cmds.BatchSetTTL("user:*", 60*time.Second)()
		result := msg.(types.BatchTTLSetMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Count != 3 {
			t.Errorf("Count = %d, want 3", result.Count)
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.BatchSetTTL("*", time.Second)()
		result := msg.(types.BatchTTLSetMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestEvalLuaScript(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.EvalResult = "OK"
		msg := cmds.EvalLuaScript("return 'OK'", nil)()
		result := msg.(types.LuaScriptResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Result != "OK" {
			t.Errorf("Result = %v, want %q", result.Result, "OK")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.EvalError = errors.New("script error")
		msg := cmds.EvalLuaScript("bad", nil)()
		result := msg.(types.LuaScriptResultMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.EvalLuaScript("return 1", nil)()
		result := msg.(types.LuaScriptResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestPublishMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.PublishResult = 2
		msg := cmds.PublishMessage("chan", "hello")()
		result := msg.(types.PublishResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Receivers != 2 {
			t.Errorf("Receivers = %d, want 2", result.Receivers)
		}
		if result.Channel != "chan" {
			t.Errorf("Channel = %q, want %q", result.Channel, "chan")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.PublishMessage("ch", "msg")()
		result := msg.(types.PublishResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestTestConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.TestConnectionLatency = 5 * time.Millisecond
		msg := cmds.TestConnection("localhost", 6379, "", 0)()
		result := msg.(types.ConnectionTestMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if !result.Success {
			t.Error("expected Success=true")
		}
		if result.Latency != 5*time.Millisecond {
			t.Errorf("Latency = %v, want %v", result.Latency, 5*time.Millisecond)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.TestConnectionError = errors.New("connection failed")
		msg := cmds.TestConnection("localhost", 6379, "", 0)()
		result := msg.(types.ConnectionTestMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
		if result.Success {
			t.Error("expected Success=false on error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.TestConnection("localhost", 6379, "", 0)()
		result := msg.(types.ConnectionTestMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestLoadKeyPrefixes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.KeyPrefixesResult = []string{"user:", "session:"}
		msg := cmds.LoadKeyPrefixes(":", 3)()
		result := msg.(types.TreeNodeExpandedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Children) != 2 {
			t.Errorf("expected 2 prefixes, got %d", len(result.Children))
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.LoadKeyPrefixes(":", 3)()
		result := msg.(types.TreeNodeExpandedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestCollectionErrors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testutil.FullMockRedisClient)
		execute func(*Commands) interface{}
		wantErr bool
	}{
		{
			"AddToList error",
			func(m *testutil.FullMockRedisClient) { m.RPushError = errors.New("err") },
			func(c *Commands) interface{} { return c.AddToList("k", "v")() },
			true,
		},
		{
			"AddToSet error",
			func(m *testutil.FullMockRedisClient) { m.SAddError = errors.New("err") },
			func(c *Commands) interface{} { return c.AddToSet("k", "v")() },
			true,
		},
		{
			"AddToZSet error",
			func(m *testutil.FullMockRedisClient) { m.ZAddError = errors.New("err") },
			func(c *Commands) interface{} { return c.AddToZSet("k", 1.0, "v")() },
			true,
		},
		{
			"AddToHash error",
			func(m *testutil.FullMockRedisClient) { m.HSetError = errors.New("err") },
			func(c *Commands) interface{} { return c.AddToHash("k", "f", "v")() },
			true,
		},
		{
			"AddToStream error",
			func(m *testutil.FullMockRedisClient) { m.XAddError = errors.New("err") },
			func(c *Commands) interface{} {
				return c.AddToStream("k", map[string]interface{}{"f": "v"})()
			},
			true,
		},
		{
			"RemoveFromList error",
			func(m *testutil.FullMockRedisClient) { m.LRemError = errors.New("err") },
			func(c *Commands) interface{} { return c.RemoveFromList("k", "v")() },
			true,
		},
		{
			"RemoveFromSet error",
			func(m *testutil.FullMockRedisClient) { m.SRemError = errors.New("err") },
			func(c *Commands) interface{} { return c.RemoveFromSet("k", "v")() },
			true,
		},
		{
			"RemoveFromZSet error",
			func(m *testutil.FullMockRedisClient) { m.ZRemError = errors.New("err") },
			func(c *Commands) interface{} { return c.RemoveFromZSet("k", "v")() },
			true,
		},
		{
			"RemoveFromHash error",
			func(m *testutil.FullMockRedisClient) { m.HDelError = errors.New("err") },
			func(c *Commands) interface{} { return c.RemoveFromHash("k", "f")() },
			true,
		},
		{
			"RemoveFromStream error",
			func(m *testutil.FullMockRedisClient) { m.XDelError = errors.New("err") },
			func(c *Commands) interface{} { return c.RemoveFromStream("k", "1-0")() },
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := testutil.NewFullMockRedisClient()
			tt.setup(mock)
			cmds := NewCommands(nil, mock)
			result := tt.execute(cmds)

			// All collection messages have an Err field
			switch msg := result.(type) {
			case types.ItemAddedToCollectionMsg:
				if tt.wantErr && msg.Err == nil {
					t.Error("expected error")
				}
			case types.ItemRemovedFromCollectionMsg:
				if tt.wantErr && msg.Err == nil {
					t.Error("expected error")
				}
			default:
				t.Errorf("unexpected message type: %T", result)
			}
		})
	}
}

func TestAutoConnect(t *testing.T) {
	t.Run("success standard", func(t *testing.T) {
		cmds, mock := newMockCmds()
		conn := types.Connection{Host: "localhost", Port: 6379, DB: 0}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(mock.Calls) == 0 || mock.Calls[0] != "Connect" {
			t.Errorf("expected Connect call, got %v", mock.Calls)
		}
	})

	t.Run("success cluster", func(t *testing.T) {
		cmds, mock := newMockCmds()
		conn := types.Connection{Host: "localhost", Port: 7000, UseCluster: true}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(mock.Calls) == 0 || mock.Calls[0] != "ConnectCluster" {
			t.Errorf("expected ConnectCluster call, got %v", mock.Calls)
		}
	})

	t.Run("success TLS", func(t *testing.T) {
		cmds, mock := newMockCmds()
		conn := types.Connection{
			Host:   "localhost",
			Port:   6380,
			UseTLS: true,
			TLSConfig: &types.TLSConfig{
				InsecureSkipVerify: true,
			},
		}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(mock.Calls) == 0 || mock.Calls[0] != "ConnectWithTLS" {
			t.Errorf("expected ConnectWithTLS call, got %v", mock.Calls)
		}
	})

	t.Run("TLS without config falls back to standard", func(t *testing.T) {
		cmds, mock := newMockCmds()
		conn := types.Connection{Host: "localhost", Port: 6379, UseTLS: true}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(mock.Calls) == 0 || mock.Calls[0] != "Connect" {
			t.Errorf("expected Connect (fallback) call, got %v", mock.Calls)
		}
	})

	t.Run("TLS bad cert file", func(t *testing.T) {
		cmds, _ := newMockCmds()
		conn := types.Connection{
			Host:   "localhost",
			Port:   6380,
			UseTLS: true,
			TLSConfig: &types.TLSConfig{
				CertFile: "/nonexistent/cert.pem",
				KeyFile:  "/nonexistent/key.pem",
			},
		}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error for bad TLS cert file")
		}
	})

	t.Run("connect error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectError = errors.New("connection refused")
		conn := types.Connection{Host: "localhost", Port: 6379}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("cluster error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectClusterError = errors.New("cluster unavailable")
		conn := types.Connection{Host: "localhost", Port: 7000, UseCluster: true}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("TLS connect error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.ConnectWithTLSError = errors.New("TLS handshake failed")
		conn := types.Connection{
			Host:      "localhost",
			Port:      6380,
			UseTLS:    true,
			TLSConfig: &types.TLSConfig{InsecureSkipVerify: true},
		}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		conn := types.Connection{Host: "localhost", Port: 6379}
		msg := cmds.AutoConnect(conn)()
		result := msg.(types.ConnectedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestNewCommandsFromContainer(t *testing.T) {
	mock := testutil.NewFullMockRedisClient()
	cfg := testutil.NewTestConfig(t)
	container := &service.Container{Config: cfg, Redis: mock}
	cmds := NewCommandsFromContainer(container)
	if cmds.config != cfg {
		t.Error("config not set from container")
	}
	if cmds.redis != mock {
		t.Error("redis not set from container")
	}
}
