package client

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aluoty/probe.git/internal/config"
)

// SaveCookies appends Set-Cookie headers from the response to the jar file.
func SaveCookies(cfg *config.Config, resp *http.Response) error {
	if cfg.CookieJar == "" {
		return nil
	}

	setCookies := resp.Header.Values("Set-Cookie")
	if len(setCookies) == 0 {
		return nil
	}

	f, err := os.OpenFile(cfg.CookieJar, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open cookie jar: %w", err)
	}
	defer f.Close()

	for _, c := range setCookies {
		if _, err := fmt.Fprintln(f, strings.TrimSpace(c)); err != nil {
			return fmt.Errorf("write cookie jar: %w", err)
		}
	}
	return nil
}
