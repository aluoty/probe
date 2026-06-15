package download

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aluoty/probe.git/internal/config"
)

// ResolveOutputPath picks the destination file from -o, -O, or stdout.
func ResolveOutputPath(cfg *config.Config, resp *http.Response) (string, bool, error) {
	var name string
	switch {
	case cfg.Output != "":
		name = cfg.Output
	case cfg.RemoteName:
		var err error
		name, err = RemoteFilename(cfg.URLs[0], resp)
		if err != nil {
			return "", false, err
		}
	default:
		return "", false, nil
	}

	if cfg.DirPrefix != "" {
		name = filepath.Join(cfg.DirPrefix, name)
	}
	return name, true, nil
}

// JoinDir joins a directory prefix with a filename.
func JoinDir(dir, name string) string {
	return filepath.Join(dir, name)
}

// RemoteFilename derives a local filename from Content-Disposition or the URL path.
func RemoteFilename(rawURL string, resp *http.Response) (string, error) {
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if name := parseContentDisposition(cd); name != "" {
			return name, nil
		}
	}
	return remoteFilenameFromURL(rawURL)
}

func parseContentDisposition(header string) string {
	parts := strings.Split(header, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "filename=") {
			name := strings.TrimPrefix(part, "filename=")
			name = strings.TrimPrefix(name, "FILENAME=")
			return strings.Trim(name, `"`)
		}
	}
	return ""
}

func remoteFilenameFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	base := path.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return "index.html", nil
	}
	return base, nil
}

// PrepareResume checks for a partial file and adds a Range header before the request.
func PrepareResume(cfg *config.Config) error {
	if !cfg.Continue {
		return nil
	}

	dest := cfg.Output
	if dest == "" && cfg.RemoteName {
		name, err := remoteFilenameFromURL(cfg.URLs[0])
		if err != nil {
			return err
		}
		dest = name
	}
	if cfg.DirPrefix != "" {
		dest = filepath.Join(cfg.DirPrefix, dest)
	}
	if dest == "" {
		return fmt.Errorf("-c requires -o or -O")
	}

	info, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat partial file: %w", err)
	}

	cfg.ResumeOffset = info.Size()
	cfg.ResumeFile = dest
	cfg.Headers = append(cfg.Headers, fmt.Sprintf("Range: bytes=%d-", cfg.ResumeOffset))

	if !cfg.Silent {
		fmt.Fprintf(os.Stderr, "resuming download at byte %d\n", cfg.ResumeOffset)
	}
	return nil
}

// WriteToFile saves the response body to disk, optionally resuming.
func WriteToFile(cfg *config.Config, resp *http.Response, dest string) (int64, error) {
	if cfg.NoClobber {
		if _, err := os.Stat(dest); err == nil {
			if !cfg.Silent {
				fmt.Fprintf(os.Stderr, "skipping existing file %s\n", dest)
			}
			io.Copy(io.Discard, resp.Body)
			return 0, nil
		}
	}

	if dir := filepath.Dir(dest); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 0, fmt.Errorf("create directory: %w", err)
		}
	}

	if cfg.Continue && cfg.ResumeOffset > 0 && dest == cfg.ResumeFile {
		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("server does not support resume (status %s)", resp.Status)
		}
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return 0, fmt.Errorf("open output file: %w", err)
		}
		defer out.Close()

		written, err := io.Copy(out, resp.Body)
		if err != nil {
			return 0, fmt.Errorf("write file: %w", err)
		}

		total := written + cfg.ResumeOffset
		if !cfg.Silent {
			fmt.Fprintf(os.Stderr, "saved %d bytes to %s\n", total, dest)
		}
		return total, nil
	}

	out, err := os.Create(dest)
	if err != nil {
		return 0, fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("write file: %w", err)
	}

	if !cfg.Silent {
		fmt.Fprintf(os.Stderr, "saved %d bytes to %s\n", written, dest)
	}
	return written, nil
}
