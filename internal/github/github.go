package github

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aluoty/probe.git/internal/client"
	"github.com/aluoty/probe.git/internal/config"
)

// Download fetches a GitHub repository ZIP archive and extracts it locally.
// GitHub ZIPs never include a .git directory.
func Download(cfg *config.Config) error {
	owner, repo, ref, err := parseSpec(cfg.GitHub)
	if err != nil {
		return err
	}

	dest := cfg.GitHubDir
	if dest == "" {
		dest = repo
	}
	if cfg.DirPrefix != "" {
		dest = filepath.Join(cfg.DirPrefix, dest)
	}

	zipURL := archiveURL(owner, repo, ref)
	if !cfg.Silent {
		fmt.Fprintf(os.Stderr, "fetching %s\n", zipURL)
	}

	reqCfg := cfg.Clone(zipURL)
	reqCfg.Method = "GET"
	reqCfg.MethodSet = true
	reqCfg.FollowRedirect = true

	start := time.Now()
	resp, err := client.Fetch(reqCfg, &client.RequestBody{})
	if err != nil {
		return fmt.Errorf("download repo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("GitHub returned HTTP %s (check owner/repo/ref)", resp.Status)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(strings.ToLower(ct), "zip") && !strings.Contains(strings.ToLower(ct), "octet-stream") {
		return fmt.Errorf("unexpected content type %q (check owner/repo/ref)", ct)
	}

	tmp, err := os.CreateTemp("", "probe-github-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	written, err := io.Copy(tmp, resp.Body)
	if err != nil {
		tmp.Close()
		return fmt.Errorf("save zip: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := extractZip(tmpPath, dest); err != nil {
		return err
	}

	if cfg.Verbose {
		client.LogResponseSummary(os.Stderr, resp, time.Since(start), written)
	}

	if !cfg.Silent {
		fmt.Fprintf(os.Stderr, "extracted %s/%s@%s -> %s\n", owner, repo, ref, dest)
	}
	return nil
}

func parseSpec(spec string) (owner, repo, ref string, err error) {
	ref = "main"
	spec = strings.TrimSpace(spec)
	spec = strings.TrimPrefix(spec, "https://github.com/")
	spec = strings.TrimPrefix(spec, "http://github.com/")
	spec = strings.TrimSuffix(spec, ".git")
	spec = strings.TrimSuffix(spec, "/")

	if spec == "" {
		return "", "", "", fmt.Errorf("empty --github value (expected owner/repo[@ref])")
	}

	if at := strings.LastIndex(spec, "@"); at > 0 {
		ref = spec[at+1:]
		spec = spec[:at]
	}

	parts := strings.Split(spec, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("invalid --github %q (expected owner/repo[@ref])", spec)
	}
	return parts[0], parts[1], ref, nil
}

func archiveURL(owner, repo, ref string) string {
	if strings.HasPrefix(ref, "refs/") {
		return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s", owner, repo, ref)
	}
	if len(ref) >= 7 && isHex(ref) {
		return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s", owner, repo, ref)
	}
	if strings.HasPrefix(ref, "v") || strings.Contains(ref, ".") {
		return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/tags/%s", owner, repo, ref)
	}
	return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/heads/%s", owner, repo, ref)
}

func isHex(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func extractZip(zipPath, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		rel, ok := stripGitHubRoot(f.Name)
		if !ok || rel == "" {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(rel))
		target, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		destAbs, err := filepath.Abs(destDir)
		if err != nil {
			return err
		}
		if target != destAbs && !strings.HasPrefix(target, destAbs+string(os.PathSeparator)) {
			return fmt.Errorf("zip slip blocked: %s", rel)
		}

		if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/") {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("mkdir parent: %w", err)
		}

		if err := extractFile(f, target); err != nil {
			return err
		}
	}
	return nil
}

// GitHub archives contain a single top-level folder (repo-ref/); strip it.
func stripGitHubRoot(name string) (string, bool) {
	name = strings.TrimPrefix(name, "./")
	parts := strings.SplitN(name, "/", 2)
	if len(parts) < 2 {
		return "", false
	}
	return parts[1], true
}

func extractFile(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open zip entry: %w", err)
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode(f))
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return fmt.Errorf("extract %s: %w", target, err)
	}
	return nil
}

func fileMode(f *zip.File) os.FileMode {
	if m := f.Mode(); m != 0 {
		return m
	}
	return 0o644
}
