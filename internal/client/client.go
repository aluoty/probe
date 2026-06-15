package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aluoty/probe.git/internal/config"
)

const maxRedirects = 10

// Fetch performs an HTTP request with retries and optional redirect following.
func Fetch(cfg *config.Config, body io.Reader) (*http.Response, error) {
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !cfg.FollowRedirect {
				return http.ErrUseLastResponse
			}
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			if cfg.Verbose && !cfg.Silent {
				fmt.Fprintf(os.Stderr, "-> %s\n", req.URL)
			}
			return nil
		},
	}

	var lastErr error
	attempts := cfg.Retry + 1

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			if cfg.Verbose && !cfg.Silent {
				fmt.Fprintf(os.Stderr, "retry %d/%d after %s\n", attempt, cfg.Retry, backoff)
			}
			time.Sleep(backoff)
		}

		req, err := buildRequest(cfg, body)
		if err != nil {
			return nil, err
		}

		if cfg.Verbose && !cfg.Silent {
			fmt.Fprintf(os.Stderr, "> %s %s\n", req.Method, req.URL)
			for k, vals := range req.Header {
				for _, v := range vals {
					fmt.Fprintf(os.Stderr, "> %s: %s\n", k, v)
				}
			}
			fmt.Fprintln(os.Stderr)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		return resp, nil
	}

	return nil, lastErr
}

func buildRequest(cfg *config.Config, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(cfg.Method, cfg.URL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", cfg.UserAgent)

	for _, h := range cfg.Headers {
		key, val, ok := strings.Cut(h, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header %q (expected Key: Value)", h)
		}
		req.Header.Set(strings.TrimSpace(key), strings.TrimSpace(val))
	}

	if cfg.BasicAuth != "" {
		user, pass, ok := strings.Cut(cfg.BasicAuth, ":")
		if !ok {
			return nil, fmt.Errorf("invalid basic auth %q (expected user:password)", cfg.BasicAuth)
		}
		req.SetBasicAuth(user, pass)
	}

	return req, nil
}

// LoadRequestBody resolves -d data, including @file and @- stdin references.
func LoadRequestBody(cfg *config.Config) (io.Reader, error) {
	if cfg.Data == "" {
		return nil, nil
	}

	data := cfg.Data
	if strings.HasPrefix(data, "@") {
		ref := strings.TrimPrefix(data, "@")
		if ref == "-" {
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("read stdin: %w", err)
			}
			return strings.NewReader(string(content)), nil
		}
		content, err := os.ReadFile(ref)
		if err != nil {
			return nil, fmt.Errorf("read data file: %w", err)
		}
		return strings.NewReader(string(content)), nil
	}

	return strings.NewReader(data), nil
}
