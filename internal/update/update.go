package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
)

const (
	githubOwner = "smichalabs"
	// Releases are published to a dedicated repo so that goreleaser artifacts
	// (signed binaries, SBOMs, checksums) live separately from source-side
	// release-please tags. The install script uses this same repo, and using
	// it here keeps the update self-replace path consistent with the install
	// path.
	githubRepo = "britivectl-releases"
)

// CheckLatest fetches the latest release from GitHub and returns the version,
// whether it's newer than currentVersion, and any error.
// The context controls cancellation; callers should set a reasonable timeout.
func CheckLatest(ctx context.Context, currentVersion string) (string, bool, error) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, githubOwner, githubRepo)
	if err != nil {
		return "", false, fmt.Errorf("fetching latest release: %w", err)
	}

	latest := strings.TrimPrefix(release.GetTagName(), "v")
	current := strings.TrimPrefix(currentVersion, "v")

	// Never prompt to update dev or alpha builds
	isNewer := latest != current && current != "0.0.1-alpha" && current != "dev"
	return latest, isNewer, nil
}

// DoUpdate downloads the specified release version and replaces the running binary.
// The context controls cancellation for the download and extraction.
func DoUpdate(ctx context.Context, version string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
	}
	arch, ok := archMap[goarch]
	if !ok {
		return fmt.Errorf("unsupported architecture: %s", goarch)
	}

	// Asset name follows goreleaser convention: bctl_Darwin_arm64.tar.gz
	osName := strings.ToUpper(goos[:1]) + goos[1:]
	assetName := fmt.Sprintf("bctl_%s_%s.tar.gz", osName, arch)
	checksumAsset := "checksums.txt"

	baseURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s", githubOwner, githubRepo, version)
	downloadURL := baseURL + "/" + assetName
	checksumURL := baseURL + "/" + checksumAsset

	tmpDir, err := os.MkdirTemp("", "bctl-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tarPath := filepath.Join(tmpDir, assetName)
	if err := downloadFile(ctx, downloadURL, tarPath); err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}

	// Verify checksum if available
	checksumPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(ctx, checksumURL, checksumPath); err == nil {
		if err := verifyChecksum(tarPath, checksumPath, assetName); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	} else {
		fmt.Printf("Warning: could not download checksums: %v\n", err)
	}

	// Extract the bctl binary from the tar.gz
	binaryPath := filepath.Join(tmpDir, "bctl")
	if err := extractBinary(tarPath, binaryPath); err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// Find and replace the running binary
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	dir := filepath.Dir(self)
	tmp, err := os.CreateTemp(dir, "bctl-new-*")
	if err != nil {
		return fmt.Errorf("creating temp binary: %w", err)
	}
	tmpBin := tmp.Name()
	defer os.Remove(tmpBin)

	newBin, err := os.Open(binaryPath) //nolint:gosec // binaryPath is a temp file we downloaded to a known location
	if err != nil {
		return fmt.Errorf("opening new binary: %w", err)
	}
	defer newBin.Close()

	if _, err := io.Copy(tmp, newBin); err != nil {
		tmp.Close()
		return fmt.Errorf("copying new binary: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpBin, 0o755); err != nil { //nolint:gosec // 0755 is required for an executable binary
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpBin, self); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

func downloadFile(ctx context.Context, url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest) //nolint:gosec // dest is a temp dir path we control
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}

func verifyChecksum(filePath, checksumPath, assetName string) error {
	data, err := os.ReadFile(checksumPath) //nolint:gosec // checksumPath is a temp file we wrote
	if err != nil {
		return err
	}

	var expectedHash string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, assetName) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				expectedHash = parts[0]
			}
			break
		}
	}
	if expectedHash == "" {
		return fmt.Errorf("checksum for %s not found in checksums file", assetName)
	}

	f, err := os.Open(filePath) //nolint:gosec // filePath is a temp file we downloaded
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualHash := fmt.Sprintf("%x", h.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

// extractBinary extracts the bctl binary from a .tar.gz archive.
func extractBinary(tarPath, destPath string) error {
	f, err := os.Open(tarPath) //nolint:gosec // tarPath is a temp file downloaded from a verified GitHub release
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}
		if filepath.Base(hdr.Name) == "bctl" && hdr.Typeflag == tar.TypeReg {
			out, err := os.Create(destPath) //nolint:gosec // destPath is a temp file in a dir we created
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			const maxBinarySize = 100 << 20 // 100 MB sanity cap
			if _, err := io.Copy(out, io.LimitReader(tr, maxBinarySize)); err != nil {
				out.Close()
				return fmt.Errorf("extracting binary: %w", err)
			}
			out.Close()
			return nil
		}
	}
	return fmt.Errorf("bctl binary not found in archive")
}
