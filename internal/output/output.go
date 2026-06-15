package output

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aluoty/probe.git/internal/config"
	"github.com/aluoty/probe.git/internal/download"
)

// HandleResponse formats and writes the HTTP response according to config.
func HandleResponse(cfg *config.Config, resp *http.Response) error {
	if cfg.Spider {
		return handleSpider(cfg, resp)
	}

	dest, toFile, err := download.ResolveOutputPath(cfg, resp)
	if err != nil {
		return err
	}

	if cfg.FailOnError && isHTTPError(resp.StatusCode) {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	if toFile {
		_, err := download.WriteToFile(cfg, resp, dest)
		return err
	}

	if cfg.IncludeHeaders {
		if err := WriteHeaders(os.Stdout, resp); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout)
	}

	if cfg.Method != "HEAD" {
		if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
			return fmt.Errorf("write body: %w", err)
		}
	} else if !cfg.IncludeHeaders && !cfg.Silent {
		fmt.Println(resp.Status)
	}

	return nil
}

func handleSpider(cfg *config.Config, resp *http.Response) error {
	if !cfg.Silent {
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			fmt.Fprintf(os.Stderr, "URL exists: %s (%s)\n", cfg.URL, resp.Status)
		} else {
			fmt.Fprintf(os.Stderr, "URL missing or error: %s (%s)\n", cfg.URL, resp.Status)
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
