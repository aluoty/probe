package output

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aluoty/probe.git/internal/client"
	"github.com/aluoty/probe.git/internal/config"
	"github.com/aluoty/probe.git/internal/download"
)

// HandleResponse formats and writes the HTTP response according to config.
func HandleResponse(cfg *config.Config, resp *http.Response, elapsed time.Duration) error {
	if err := client.SaveCookies(cfg, resp); err != nil {
		return err
	}

	if cfg.DumpHeaders != "" {
		if err := dumpHeaders(cfg.DumpHeaders, resp); err != nil {
			return err
		}
	}

	if cfg.Spider {
		err := handleSpider(cfg, resp)
		logVerbose(cfg, resp, elapsed, -1, err)
		return err
	}

	if cfg.FailOnError && isHTTPError(resp.StatusCode) {
		err := fmt.Errorf("HTTP %s", resp.Status)
		logVerbose(cfg, resp, elapsed, -1, err)
		return err
	}

	dest, toFile, err := download.ResolveOutputPath(cfg, resp)
	if err != nil {
		return err
	}

	var peek []byte
	if !toFile && cfg.Method != "HEAD" {
		peek, err = download.PeekBodyDefault(resp)
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		if download.ShouldSaveBinary(resp, peek) {
			name, err := download.RemoteFilename(cfg.URLs[0], resp)
			if err != nil {
				return err
			}
			dest = name
			if cfg.DirPrefix != "" {
				dest = download.JoinDir(cfg.DirPrefix, name)
			}
			toFile = true
			if !cfg.Silent {
				fmt.Fprintf(os.Stderr, "binary response, saving to %s\n", dest)
			}
		}
	}

	var bodyBytes int64
	var handleErr error

	switch {
	case toFile:
		bodyBytes, handleErr = download.WriteToFile(cfg, resp, dest)
	case cfg.IncludeHeaders:
		if err := WriteHeaders(os.Stdout, resp); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout)
		if cfg.Method != "HEAD" {
			bodyBytes, handleErr = copyOut(os.Stdout, resp.Body)
		}
	case cfg.Method != "HEAD":
		bodyBytes, handleErr = copyOut(os.Stdout, resp.Body)
	default:
		if !cfg.Silent {
			fmt.Println(resp.Status)
		}
	}

	if handleErr != nil {
		logVerbose(cfg, resp, elapsed, bodyBytes, handleErr)
		return handleErr
	}

	logVerbose(cfg, resp, elapsed, bodyBytes, nil)
	return nil
}

func copyOut(w io.Writer, r io.Reader) (int64, error) {
	n, err := io.Copy(w, r)
	if err != nil {
		return n, fmt.Errorf("write body: %w", err)
	}
	return n, nil
}

func logVerbose(cfg *config.Config, resp *http.Response, elapsed time.Duration, bodyBytes int64, err error) {
	if !cfg.Verbose {
		return
	}
	client.LogResponseSummary(os.Stderr, resp, elapsed, bodyBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "* error: %v\n", err)
	}
}

func dumpHeaders(path string, resp *http.Response) error {
	if path == "-" {
		return WriteHeaders(os.Stdout, resp)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create header file: %w", err)
	}
	defer f.Close()

	return WriteHeaders(f, resp)
}

func handleSpider(cfg *config.Config, resp *http.Response) error {
	if !cfg.Silent {
		url := cfg.URLs[0]
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			fmt.Fprintf(os.Stderr, "URL exists: %s (%s)\n", url, resp.Status)
		} else {
			fmt.Fprintf(os.Stderr, "URL missing or error: %s (%s)\n", url, resp.Status)
		}
	}
	if cfg.FailOnError && isHTTPError(resp.StatusCode) {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	return nil
}

// WriteHeaders writes the HTTP status line and response headers to w.
func WriteHeaders(w io.Writer, resp *http.Response) error {
	statusLine := fmt.Sprintf("HTTP/%d.%d %s", resp.ProtoMajor, resp.ProtoMinor, resp.Status)
	if _, err := fmt.Fprintln(w, statusLine); err != nil {
		return err
	}
	for k, vals := range resp.Header {
		for _, v := range vals {
			if _, err := fmt.Fprintf(w, "%s: %s\n", k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func isHTTPError(status int) bool {
	return status >= 400
}
