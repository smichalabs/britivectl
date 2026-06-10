package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsVersionNewer locks down the version comparison so we never again
// prompt users to "update" to an older tag. The previous implementation
// used string inequality (`latest != current`), which happily reported
// 0.9.1 as "newer than" 0.10.2 for a user on the higher patch. This is
// the regression test for that bug.
func TestIsVersionNewer(t *testing.T) {
	cases := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		// The bug we just fixed: 0.9.1 must NOT be reported as newer than
		// 0.10.2 (the user's actual installed version). Lexical comparison
		// would say yes; semver comparison says no.
		{name: "9.1 not newer than 10.2 (the reported bug)", latest: "0.9.1", current: "0.10.2", want: false},

		// Direction sanity checks across major / minor / patch.
		{name: "patch bump is newer", latest: "0.10.3", current: "0.10.2", want: true},
		{name: "minor bump is newer", latest: "0.11.0", current: "0.10.2", want: true},
		{name: "major bump is newer", latest: "1.0.0", current: "0.10.2", want: true},
		{name: "patch downgrade is not newer", latest: "0.10.1", current: "0.10.2", want: false},
		{name: "minor downgrade is not newer", latest: "0.9.0", current: "0.10.2", want: false},
		{name: "major downgrade is not newer", latest: "0.10.2", current: "1.0.0", want: false},

		// Equal versions must never trigger an update prompt.
		{name: "equal versions are not newer", latest: "0.10.2", current: "0.10.2", want: false},

		// Tag prefix tolerance -- callers may pass either form.
		{name: "v-prefixed latest is normalized", latest: "v0.10.3", current: "0.10.2", want: true},
		{name: "v-prefixed current is normalized", latest: "0.10.3", current: "v0.10.2", want: true},
		{name: "both v-prefixed", latest: "v0.10.3", current: "v0.10.2", want: true},

		// Dev / placeholder builds should never see an update prompt --
		// they're local builds without a meaningful tag to compare against.
		{name: "dev current returns false", latest: "0.10.3", current: "dev", want: false},
		{name: "alpha placeholder returns false", latest: "0.10.3", current: "0.0.1-alpha", want: false},
		{name: "empty current returns false", latest: "0.10.3", current: "", want: false},
		{name: "dev-suffixed current returns false", latest: "0.10.3", current: "0.10.2-dev", want: false},

		// Non-semver garbage should be treated as "not newer" so a broken
		// GitHub response can't trick the binary into a self-update.
		{name: "non-semver latest is rejected", latest: "garbage", current: "0.10.2", want: false},
		{name: "non-semver current is rejected", latest: "0.10.3", current: "garbage", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isVersionNewer(tc.latest, tc.current); got != tc.want {
				t.Errorf("isVersionNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
			}
		})
	}
}

// TestGithubRepoConstant pins the constant so a future refactor can't
// silently flip the update path back at the source repo. install.sh and
// the in-binary path must agree on which repo serves the canonical
// "latest release" answer; if they diverge again, users get suggested
// downgrades like the original bug.
func TestGithubRepoConstant(t *testing.T) {
	if githubRepo != "britivectl-releases" {
		t.Errorf("githubRepo = %q, want \"britivectl-releases\" -- the source repo's /releases/latest is unreliable because release-please leaves drafts there", githubRepo)
	}
}

// fakeGitHubServer stands up an httptest server that responds to the one
// endpoint go-github hits for GetLatestRelease and returns a release with
// the given tag. Sets githubBaseURL to the test server for the duration of
// the test.
func fakeGitHubServer(t *testing.T, tag string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := fmt.Sprintf("/repos/%s/%s/releases/latest", githubOwner, githubRepo)
		if r.URL.Path != want {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tag_name":%q}`, tag)
	}))
	t.Cleanup(srv.Close)

	prev := githubBaseURL
	// go-github requires the base URL to end with a trailing slash.
	githubBaseURL = srv.URL + "/"
	t.Cleanup(func() { githubBaseURL = prev })
}

func TestCheckLatest_ReportsNewerVersionWhenUpstreamIsAhead(t *testing.T) {
	fakeGitHubServer(t, "v0.10.3")
	latest, newer, err := CheckLatest(context.Background(), "0.10.2")
	if err != nil {
		t.Fatalf("CheckLatest: %v", err)
	}
	if latest != "0.10.3" {
		t.Errorf("latest = %q, want 0.10.3", latest)
	}
	if !newer {
		t.Error("expected newer=true when upstream is ahead")
	}
}

// TestCheckLatest_DoesNotSuggestDowngrade is the regression test for the
// reported bug: source repo returned 0.9.1 to a user on 0.10.2 and bctl
// said "new version available". With semver compare CheckLatest must
// report newer=false even when the upstream tag is a real semver that is
// just lower than the user's installed version.
func TestCheckLatest_DoesNotSuggestDowngrade(t *testing.T) {
	fakeGitHubServer(t, "v0.9.1")
	latest, newer, err := CheckLatest(context.Background(), "0.10.2")
	if err != nil {
		t.Fatalf("CheckLatest: %v", err)
	}
	if latest != "0.9.1" {
		t.Errorf("latest = %q, want 0.9.1", latest)
	}
	if newer {
		t.Error("expected newer=false when upstream tag is older than installed version")
	}
}

func TestCheckLatest_EqualVersionsAreNotNewer(t *testing.T) {
	fakeGitHubServer(t, "v0.10.2")
	_, newer, err := CheckLatest(context.Background(), "0.10.2")
	if err != nil {
		t.Fatalf("CheckLatest: %v", err)
	}
	if newer {
		t.Error("expected newer=false when versions match")
	}
}

func TestCheckLatest_ServerErrorBubblesUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	prev := githubBaseURL
	githubBaseURL = srv.URL + "/"
	t.Cleanup(func() { githubBaseURL = prev })

	if _, _, err := CheckLatest(context.Background(), "0.10.2"); err == nil {
		t.Error("expected error when upstream returns 500, got nil")
	}
}

func TestDownloadFile_WritesContent(t *testing.T) {
	body := []byte("hello-bctl")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "out")
	if err := downloadFile(context.Background(), srv.URL, dest); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("downloaded content = %q, want %q", got, body)
	}
}

func TestDownloadFile_NonOKStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "out")
	err := downloadFile(context.Background(), srv.URL, dest)
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error %q should mention the HTTP status code", err)
	}
}

func TestVerifyChecksum_MatchAndMismatch(t *testing.T) {
	dir := t.TempDir()
	asset := "bctl_Linux_amd64.tar.gz"

	// Write a synthetic asset file and compute its real sha256 so we can
	// build a checksums.txt the same way goreleaser does.
	assetPath := filepath.Join(dir, asset)
	content := []byte("not really a tarball but verifyChecksum doesn't care")
	if err := os.WriteFile(assetPath, content, 0o600); err != nil {
		t.Fatalf("writing asset: %v", err)
	}
	sum := sha256.Sum256(content)
	checksums := fmt.Sprintf("%x  %s\nother  unrelated-asset.tar.gz\n", sum, asset)
	checksumsPath := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte(checksums), 0o600); err != nil {
		t.Fatalf("writing checksums: %v", err)
	}

	if err := verifyChecksum(assetPath, checksumsPath, asset); err != nil {
		t.Errorf("verifyChecksum should accept a matching hash, got %v", err)
	}

	// Now corrupt the asset and confirm we fail the check.
	if err := os.WriteFile(assetPath, []byte("tampered"), 0o600); err != nil {
		t.Fatalf("rewriting asset: %v", err)
	}
	if err := verifyChecksum(assetPath, checksumsPath, asset); err == nil {
		t.Error("verifyChecksum should reject a mismatched hash")
	}
}

// makeTarGz writes a gzip-compressed tar archive containing one entry per
// (name, body) pair. Used to drive extractBinary tests without depending on
// a real goreleaser tarball.
func makeTarGz(t *testing.T, dest string, entries map[string][]byte) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range entries {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatalf("tar body: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gz close: %v", err)
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestExtractBinary_PullsBctlFromArchive(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "bctl_Linux_amd64.tar.gz")
	binBody := []byte("#!/bin/sh\necho fake-bctl\n")
	makeTarGz(t, tarPath, map[string][]byte{
		"LICENSE": []byte("MIT"),
		"bctl":    binBody,
		"README":  []byte("readme"),
	})

	dest := filepath.Join(dir, "extracted")
	if err := extractBinary(tarPath, dest); err != nil {
		t.Fatalf("extractBinary: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(binBody) {
		t.Errorf("extracted body = %q, want %q", got, binBody)
	}
}

func TestExtractBinary_MissingBctlEntryReturnsError(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "bctl_Linux_amd64.tar.gz")
	makeTarGz(t, tarPath, map[string][]byte{
		"LICENSE": []byte("MIT"),
		"README":  []byte("no binary here"),
	})

	dest := filepath.Join(dir, "extracted")
	err := extractBinary(tarPath, dest)
	if err == nil {
		t.Fatal("expected error when bctl entry is absent, got nil")
	}
	if !strings.Contains(err.Error(), "bctl") {
		t.Errorf("error %q should mention the missing entry", err)
	}
}

func TestVerifyChecksum_AssetMissingFromChecksums(t *testing.T) {
	dir := t.TempDir()
	assetPath := filepath.Join(dir, "missing.tar.gz")
	if err := os.WriteFile(assetPath, []byte("x"), 0o600); err != nil {
		t.Fatalf("writing asset: %v", err)
	}
	checksumsPath := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte("aaaa  some-other-asset.tar.gz\n"), 0o600); err != nil {
		t.Fatalf("writing checksums: %v", err)
	}
	if err := verifyChecksum(assetPath, checksumsPath, "missing.tar.gz"); err == nil {
		t.Error("verifyChecksum should error when asset is not in checksums file")
	}
}
