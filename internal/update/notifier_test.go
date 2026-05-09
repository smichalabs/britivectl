package update

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixedClock returns a deterministic time.Now stand-in.
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// envMap turns a static map into the Env hook signature the Notifier wants.
// Tests pass nil for "no env vars set".
func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// newTestNotifier builds a Notifier with a temp cache file and clean defaults
// so each test starts from a known empty state.
func newTestNotifier(t *testing.T) *Notifier {
	t.Helper()
	dir := t.TempDir()
	return &Notifier{
		CacheFile:  filepath.Join(dir, "update_check.json"),
		CurrentVer: "v0.9.0",
		TTL:        24 * time.Hour,
		Now:        fixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)),
		Fetch:      func(_ context.Context) (string, error) { return "v0.9.5", nil },
		Env:        envMap(nil),
	}
}

func TestMaybePrintNotice_NewerCachedVersionPrintsLine(t *testing.T) {
	n := newTestNotifier(t)
	if err := saveCache(n.CacheFile, &Cache{CheckedAt: n.Now(), LatestVersion: "v0.9.5"}); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	var buf bytes.Buffer
	if !n.MaybePrintNotice(&buf, true) {
		t.Fatal("expected notice to print, got false")
	}
	if !bytes.Contains(buf.Bytes(), []byte("v0.9.5")) || !bytes.Contains(buf.Bytes(), []byte("v0.9.0")) {
		t.Errorf("notice did not include expected versions: %q", buf.String())
	}
}

func TestMaybePrintNotice_SameVersionNoNotice(t *testing.T) {
	n := newTestNotifier(t)
	n.CurrentVer = "v0.9.5"
	if err := saveCache(n.CacheFile, &Cache{CheckedAt: n.Now(), LatestVersion: "v0.9.5"}); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	var buf bytes.Buffer
	if n.MaybePrintNotice(&buf, true) {
		t.Errorf("expected no notice when versions match, got: %q", buf.String())
	}
}

func TestMaybePrintNotice_NoCacheNoNotice(t *testing.T) {
	n := newTestNotifier(t)
	var buf bytes.Buffer
	if n.MaybePrintNotice(&buf, true) {
		t.Errorf("expected no notice with missing cache, got: %q", buf.String())
	}
}

func TestMaybePrintNotice_NotTTYNoNotice(t *testing.T) {
	n := newTestNotifier(t)
	if err := saveCache(n.CacheFile, &Cache{CheckedAt: n.Now(), LatestVersion: "v0.9.5"}); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	var buf bytes.Buffer
	if n.MaybePrintNotice(&buf, false) {
		t.Error("expected no notice when stderr is not a TTY")
	}
}

func TestMaybePrintNotice_DisabledByEnv(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		{"BCTL_NO_UPDATE_CHECK set", map[string]string{"BCTL_NO_UPDATE_CHECK": "1"}},
		{"CI=true", map[string]string{"CI": "true"}},
		{"GITHUB_ACTIONS=true", map[string]string{"GITHUB_ACTIONS": "true"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := newTestNotifier(t)
			n.Env = envMap(tc.env)
			if err := saveCache(n.CacheFile, &Cache{CheckedAt: n.Now(), LatestVersion: "v0.9.5"}); err != nil {
				t.Fatalf("saveCache: %v", err)
			}
			var buf bytes.Buffer
			if n.MaybePrintNotice(&buf, true) {
				t.Errorf("expected env to suppress notice, got: %q", buf.String())
			}
		})
	}
}

func TestMaybePrintNotice_DevBuildSkipped(t *testing.T) {
	cases := []string{"dev", "v0.0.0", "v0.0.0-dev", "v1.2.3-dev"}
	for _, ver := range cases {
		t.Run(ver, func(t *testing.T) {
			n := newTestNotifier(t)
			n.CurrentVer = ver
			if err := saveCache(n.CacheFile, &Cache{CheckedAt: n.Now(), LatestVersion: "v0.9.5"}); err != nil {
				t.Fatalf("saveCache: %v", err)
			}
			var buf bytes.Buffer
			if n.MaybePrintNotice(&buf, true) {
				t.Errorf("expected dev build %q to suppress notice, got: %q", ver, buf.String())
			}
		})
	}
}

func TestRefreshIfStale_FreshCacheSkipsFetch(t *testing.T) {
	n := newTestNotifier(t)
	fetched := false
	n.Fetch = func(_ context.Context) (string, error) {
		fetched = true
		return "v0.9.5", nil
	}
	if err := saveCache(n.CacheFile, &Cache{CheckedAt: n.Now().Add(-1 * time.Hour), LatestVersion: "v0.9.4"}); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	<-n.RefreshIfStale(context.Background())
	if fetched {
		t.Error("expected fetch to be skipped on a fresh cache")
	}
}

func TestRefreshIfStale_StaleCacheTriggersFetchAndUpdates(t *testing.T) {
	n := newTestNotifier(t)
	n.Fetch = func(_ context.Context) (string, error) {
		return "v0.9.5", nil
	}
	staleTime := n.Now().Add(-25 * time.Hour)
	if err := saveCache(n.CacheFile, &Cache{CheckedAt: staleTime, LatestVersion: "v0.9.4"}); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	<-n.RefreshIfStale(context.Background())

	got, err := loadCache(n.CacheFile)
	if err != nil {
		t.Fatalf("loadCache after refresh: %v", err)
	}
	if got == nil || got.LatestVersion != "v0.9.5" {
		t.Errorf("expected cache to be updated to v0.9.5, got %+v", got)
	}
	if !got.CheckedAt.Equal(n.Now()) {
		t.Errorf("expected CheckedAt to be %v, got %v", n.Now(), got.CheckedAt)
	}
}

func TestRefreshIfStale_FetchErrorLeavesCacheUntouched(t *testing.T) {
	n := newTestNotifier(t)
	n.Fetch = func(_ context.Context) (string, error) {
		return "", errors.New("network down")
	}
	staleTime := n.Now().Add(-48 * time.Hour)
	if err := saveCache(n.CacheFile, &Cache{CheckedAt: staleTime, LatestVersion: "v0.9.4"}); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	<-n.RefreshIfStale(context.Background())

	got, err := loadCache(n.CacheFile)
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	if got.LatestVersion != "v0.9.4" {
		t.Errorf("expected cache untouched on fetch error, got %q", got.LatestVersion)
	}
}

func TestRefreshIfStale_SkipsWhenDisabled(t *testing.T) {
	n := newTestNotifier(t)
	n.Env = envMap(map[string]string{"BCTL_NO_UPDATE_CHECK": "1"})
	fetched := false
	n.Fetch = func(_ context.Context) (string, error) {
		fetched = true
		return "v0.9.5", nil
	}
	<-n.RefreshIfStale(context.Background())
	if fetched {
		t.Error("expected disabled env to suppress fetch")
	}
	if _, err := os.Stat(n.CacheFile); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected no cache file written, got stat err %v", err)
	}
}

func TestSaveAndLoadCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update_check.json")
	want := &Cache{
		CheckedAt:     time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		LatestVersion: "v0.9.2",
	}
	if err := saveCache(path, want); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	got, err := loadCache(path)
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	if got.LatestVersion != want.LatestVersion {
		t.Errorf("LatestVersion: got %q want %q", got.LatestVersion, want.LatestVersion)
	}
	if !got.CheckedAt.Equal(want.CheckedAt) {
		t.Errorf("CheckedAt: got %v want %v", got.CheckedAt, want.CheckedAt)
	}
}

func TestLoadCache_MissingFileReturnsNilNoError(t *testing.T) {
	got, err := loadCache(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Errorf("expected no error on missing file, got: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil cache, got: %+v", got)
	}
}

func TestLoadCache_CorruptFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update_check.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := loadCache(path); err == nil {
		t.Error("expected error on corrupt cache, got nil")
	}
}

func TestSaveCache_AtomicityViaTempFile(t *testing.T) {
	// Smoke test: confirm the saved file actually contains parseable JSON
	// matching the input. The atomic-rename behavior itself is hard to assert
	// without a rename hook, so this just guards the round-trip surface.
	dir := t.TempDir()
	path := filepath.Join(dir, "update_check.json")
	c := &Cache{CheckedAt: time.Now().UTC(), LatestVersion: "v0.9.2"}
	if err := saveCache(path, c); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var got Cache
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.LatestVersion != c.LatestVersion {
		t.Errorf("LatestVersion: got %q want %q", got.LatestVersion, c.LatestVersion)
	}
}
