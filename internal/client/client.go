package client

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aluoty/probe.git/internal/config"
)

// RequestBody holds the payload and any query string mutation from -G.
type RequestBody struct {
	Bytes []byte
	Query string
}

// PrepareRequestBody resolves upload/data flags into bytes or a query string.
func PrepareRequestBody(cfg *config.Config) (*RequestBody, error) {
	if cfg.UploadFile != "" {
		data, err := readAtRef(cfg.UploadFile)
		if err != nil {
			return nil, err
		}
		return &RequestBody{Bytes: data}, nil
	}

	if cfg.Data == "" {
		return &RequestBody{}, nil
	}

	if cfg.UseGET {
		query, err := dataToQuery(cfg.Data)
		if err != nil {
			return nil, err
		}
		return &RequestBody{Query: query}, nil
	}

	data, err := readAtRef(cfg.Data)
	if err != nil {
		return nil, err
	}
	return &RequestBody{Bytes: data}, nil
}

func readAtRef(ref string) ([]byte, error) {
	if strings.HasPrefix(ref, "@") {
		src := strings.TrimPrefix(ref, "@")
		if src == "-" {
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("read stdin: %w", err)
			}
			return content, nil
		}
		content, err := os.ReadFile(src)
		if err != nil {
			return nil, fmt.Errorf("read file %q: %w", src, err)
		}
		return content, nil
	}
	return []byte(ref), nil
}

func dataToQuery(data string) (string, error) {
	raw, err := readAtRef(data)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", nil
	}
	if strings.Contains(text, "=") {
		return text, nil
	}
	return url.QueryEscape(text), nil
}

// Fetch performs an HTTP request with retries and optional redirect following.
func Fetch(cfg *config.Config, body *RequestBody) (*http.Response, error) {
	targetURL := cfg.URLs[0]
	if body != nil && body.Query != "" {
		sep := "?"
		if strings.Contains(targetURL, "?") {
			sep = "&"
		}
		targetURL += sep + body.Query
	}

	httpClient := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: buildTransport(cfg),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !cfg.FollowRedirect {
				return http.ErrUseLastResponse
			}
			limit := cfg.MaxRedirects
			if limit <= 0 {
				limit = 10
			}
			if len(via) >= limit {
				return fmt.Errorf("stopped after %d redirects", limit)
			}
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "-> %s\n", req.URL)
			}
			return nil
		},
	}

	var payload []byte
	if body != nil {
		payload = body.Bytes
	}

	var lastErr error
	for attempt := 0; attempt < cfg.Retry+1; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "retry %d/%d after %s\n", attempt, cfg.Retry, backoff)
			}
			time.Sleep(backoff)
		}

		var reader io.Reader
		if len(payload) > 0 {
			reader = bytes.NewReader(payload)
		}

		req, err := buildRequest(cfg, targetURL, reader)
		if err != nil {
			return nil, err
		}

		if cfg.Verbose {
			LogVerboseRequest(cfg, req)
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

func buildTransport(cfg *config.Config) *http.Transport {
	dialer := &net.Dialer{Timeout: cfg.ConnectTimeout}
	transport := &http.Transport{
		Proxy:       http.ProxyFromEnvironment,
		DialContext: dialer.DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.Insecure,
		},
	}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return transport
}

func buildRequest(cfg *config.Config, targetURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(cfg.Method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", cfg.UserAgent)
	if cfg.Referer != "" {
		req.Header.Set("Referer", cfg.Referer)
	}

	for _, h := range cfg.Headers {
		key, val, ok := strings.Cut(h, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header %q (expected Key: Value)", h)
		}
		req.Header.Set(strings.TrimSpace(key), strings.TrimSpace(val))
	}

	if cookie, err := loadCookies(cfg); err != nil {
		return nil, err
	} else if cookie != "" {
		req.Header.Set("Cookie", cookie)
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

func loadCookies(cfg *config.Config) (string, error) {
	if cfg.Cookie == "" {
		return "", nil
	}
	if strings.HasPrefix(cfg.Cookie, "@") {
		path := strings.TrimPrefix(cfg.Cookie, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read cookies: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return cfg.Cookie, nil
}
