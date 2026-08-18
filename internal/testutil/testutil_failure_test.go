package testutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/db"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// fakeTB is a minimal testing.TB implementation that records Errorf/Fatalf
// calls so we can test failure paths of helpers without failing the real test.
// It embeds a real testing.TB so that methods like TempDir still work.
type fakeTB struct {
	testing.TB
	mu       sync.Mutex
	errorMsg string
	fatalMsg string
	helper   bool
	failed   bool
}

func (f *fakeTB) Helper() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.helper = true
}

func (f *fakeTB) Errorf(format string, _ ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errorMsg = format
	f.failed = true
}

func (f *fakeTB) Fatalf(format string, _ ...any) {
	f.mu.Lock()
	f.fatalMsg = format
	f.failed = true
	f.mu.Unlock()
	// Mimic *testing.T.Fatalf which calls runtime.Goexit.
	runtime.Goexit()
}

func (f *fakeTB) didFail() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.failed
}

// runWithFakeTB invokes fn in a goroutine with the supplied fake TB so that
// runtime.Goexit (called by Fatalf) does not terminate the parent test.
// The real *testing.T is embedded so helpers like TempDir remain functional.
func runWithFakeTB(t *testing.T, fn func(tb testing.TB)) *fakeTB {
	t.Helper()
	f := &fakeTB{TB: t}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn(f)
	}()
	wg.Wait()
	return f
}

func TestAssertEqual_Failure(t *testing.T) {
	f := runWithFakeTB(t, func(tb testing.TB) {
		AssertEqual(tb, 1, 2, "ints differ")
	})
	if !f.didFail() {
		t.Error("expected AssertEqual to mark fake TB as failed")
	}
	if f.errorMsg == "" {
		t.Error("expected Errorf to be called")
	}
}

func TestAssertNoError_Failure(t *testing.T) {
	f := runWithFakeTB(t, func(tb testing.TB) {
		AssertNoError(tb, errors.New("boom"), "should be nil")
	})
	if !f.didFail() {
		t.Error("expected AssertNoError to mark fake TB as failed")
	}
}

func TestAssertError_Failure(t *testing.T) {
	f := runWithFakeTB(t, func(tb testing.TB) {
		AssertError(tb, nil, "should not be nil")
	})
	if !f.didFail() {
		t.Error("expected AssertError to mark fake TB as failed")
	}
}

func TestAssertSliceLen_Failure(t *testing.T) {
	f := runWithFakeTB(t, func(tb testing.TB) {
		AssertSliceLen(tb, []int{1, 2}, 5, "wrong len")
	})
	if !f.didFail() {
		t.Error("expected AssertSliceLen to mark fake TB as failed")
	}
}

func TestAssertConnectionExists_NotFound(t *testing.T) {
	cfg := NewTestConfig(t)
	f := runWithFakeTB(t, func(tb testing.TB) {
		AssertConnectionExists(tb, cfg, 12345)
	})
	if !f.didFail() {
		t.Error("expected AssertConnectionExists to mark fake TB as failed")
	}
	if f.fatalMsg == "" {
		t.Error("expected Fatalf to be called")
	}
}

func TestAssertConnectionNotExists_FoundFails(t *testing.T) {
	cfg := NewTestConfig(t)
	conn := MustAddConnection(t, cfg, types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	f := runWithFakeTB(t, func(tb testing.TB) {
		AssertConnectionNotExists(tb, cfg, conn.ID)
	})
	if !f.didFail() {
		t.Error("expected AssertConnectionNotExists to fail when connection exists")
	}
}

func TestNewTestConfig_DBFailure(t *testing.T) {
	orig := dbNewConfig
	dbNewConfig = func(string) (*db.Config, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { dbNewConfig = orig })

	f := runWithFakeTB(t, func(tb testing.TB) {
		NewTestConfig(tb)
	})
	if !f.didFail() {
		t.Error("expected NewTestConfig to call Fatalf when dbNewConfig errors")
	}
}

func TestAssertConnectionExists_ListError(t *testing.T) {
	orig := listConnectionsFunc
	listConnectionsFunc = func(*db.Config) ([]types.Connection, error) {
		return nil, errors.New("list failed")
	}
	t.Cleanup(func() { listConnectionsFunc = orig })

	cfg := NewTestConfig(t)
	f := runWithFakeTB(t, func(tb testing.TB) {
		AssertConnectionExists(tb, cfg, 1)
	})
	if !f.didFail() {
		t.Error("expected AssertConnectionExists to call Fatalf when list errors")
	}
}

func TestAssertConnectionNotExists_ListError(t *testing.T) {
	orig := listConnectionsFunc
	listConnectionsFunc = func(*db.Config) ([]types.Connection, error) {
		return nil, errors.New("list failed")
	}
	t.Cleanup(func() { listConnectionsFunc = orig })

	cfg := NewTestConfig(t)
	f := runWithFakeTB(t, func(tb testing.TB) {
		AssertConnectionNotExists(tb, cfg, 1)
	})
	if !f.didFail() {
		t.Error("expected AssertConnectionNotExists to call Fatalf when list errors")
	}
}

func TestMustAddConnection_Failure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test requires Unix directory permissions")
	}
	if os.Geteuid() == 0 {
		t.Skip("test requires non-root user to enforce file permissions")
	}
	dir := t.TempDir()
	cfg, err := db.NewConfig(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	// Make the directory read-only so save() fails on AddConnection.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Logf("restore dir perms: %v", err)
		}
	})

	f := runWithFakeTB(t, func(tb testing.TB) {
		MustAddConnection(tb, cfg, types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	})
	if !f.didFail() {
		t.Error("expected MustAddConnection to fail when save errors")
	}
}
