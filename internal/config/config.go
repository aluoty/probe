package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	Name      = "probe"
	Version   = "0.3.0"
	UserAgent = Name + "/" + Version
)

// Config holds all CLI options for probe.
type Config struct {
	URLs           []string
	Method         string
	MethodSet      bool
	Headers        []string
	Data           string
	UseGET         bool
	UploadFile     string
	Output         string
	RemoteName     bool
	IncludeHeaders bool
	HeadOnly       bool
	FollowRedirect bool
	Silent         bool
	ShowErrors     bool
	Verbose        bool
	UserAgent      string
	Referer        string
	BasicAuth      string
	Proxy          string
	Insecure       bool
	Cookie         string
	CookieJar      string
	DumpHeaders    string
	DirPrefix      string
	NoClobber      bool
	Timeout        time.Duration
	ConnectTimeout time.Duration
	MaxRedirects   int
	Continue       bool
	Spider         bool
	Retry          int
	FailOnError    bool
	InputFile      string
	ShowHelp       bool
	ShowVersion    bool

	ResumeOffset int64
	ResumeFile   string
}

type flagArray []string

func (f *flagArray) String() string { return strings.Join(*f, ", ") }

func (f *flagArray) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// Parse builds a Config from CLI arguments (excluding the program name).
func Parse(args []string) (*Config, error) {
	cfg := &Config{
		UserAgent:    UserAgent,
		MaxRedirects: 10,
	}

	fs := flag.NewFlagSet(Name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&cfg.Method, "X", "GET", "HTTP method")
	var headers flagArray
	fs.Var(&headers, "H", "Request header (Key: Value)")
	fs.StringVar(&cfg.Data, "d", "", "Request body or form data")
	fs.BoolVar(&cfg.UseGET, "G", false, "Send -d data as URL query (GET)")
	fs.StringVar(&cfg.UploadFile, "T", "", "Upload file (@path or @-)")
	fs.StringVar(&cfg.Output, "o", "", "Write body to FILE")
	fs.BoolVar(&cfg.RemoteName, "O", false, "Save using remote filename")
	fs.BoolVar(&cfg.IncludeHeaders, "i", false, "Include response headers in output")
	fs.BoolVar(&cfg.HeadOnly, "I", false, "Fetch headers only")
	fs.BoolVar(&cfg.FollowRedirect, "L", false, "Follow redirects")
	fs.BoolVar(&cfg.Silent, "s", false, "Silent mode")
	fs.BoolVar(&cfg.ShowErrors, "S", false, "Show errors even in silent mode")
	fs.BoolVar(&cfg.Verbose, "v", false, "Verbose mode")
	fs.StringVar(&cfg.UserAgent, "A", UserAgent, "User-Agent")
	fs.StringVar(&cfg.Referer, "e", "", "Referer URL")
	fs.StringVar(&cfg.BasicAuth, "u", "", "Basic auth (user:password)")
	fs.StringVar(&cfg.Proxy, "x", "", "Proxy URL (http://host:port)")
	fs.BoolVar(&cfg.Insecure, "k", false, "Skip TLS certificate verification")
	fs.StringVar(&cfg.Cookie, "b", "", "Send cookies (string or @file)")
	fs.StringVar(&cfg.DumpHeaders, "D", "", "Write response headers to FILE")
	fs.StringVar(&cfg.DirPrefix, "P", "", "Save downloads under directory prefix")
	fs.BoolVar(&cfg.NoClobber, "nc", false, "Do not overwrite existing files")

	var timeoutSec float64
	fs.Float64Var(&timeoutSec, "timeout", 30, "Total request timeout in seconds")
	var connectSec float64
	fs.Float64Var(&connectSec, "connect-timeout", 10, "Connection timeout in seconds")
	fs.IntVar(&cfg.MaxRedirects, "max-redirs", 10, "Maximum redirects to follow")

	fs.BoolVar(&cfg.Continue, "c", false, "Resume partial download")
	fs.BoolVar(&cfg.Spider, "spider", false, "Check URL without downloading")
	fs.IntVar(&cfg.Retry, "retry", 0, "Retries on failure")
	fs.BoolVar(&cfg.FailOnError, "f", false, "Exit with error on HTTP 4xx/5xx")
	fs.StringVar(&cfg.InputFile, "input-file", "", "Read URLs from file (one per line)")
	fs.StringVar(&cfg.CookieJar, "cookie-jar", "", "Read/write cookie jar file")

	fs.BoolVar(&cfg.ShowHelp, "h", false, "Show help")
	fs.BoolVar(&cfg.ShowVersion, "V", false, "Show version")

	fs.Usage = func() { PrintUsage(fs.Output()) }

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.Headers = headers
	cfg.Timeout = time.Duration(timeoutSec * float64(time.Second))
	cfg.ConnectTimeout = time.Duration(connectSec * float64(time.Second))

	// Detect whether -X was explicitly passed.
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "X" {
			cfg.MethodSet = true
		}
	})

	cfg.URLs = fs.Args()
	if cfg.InputFile != "" {
		lines, err := readURLList(cfg.InputFile)
		if err != nil {
			return nil, err
		}
		cfg.URLs = append(cfg.URLs, lines...)
	}

	return cfg, cfg.validate()
}

func readURLList(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input file: %w", err)
	}
	var urls []string
	for line := range strings.Lines(string(data)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	return urls, nil
}

func (c *Config) validate() error {
	if c.ShowHelp || c.ShowVersion {
		return nil
	}
	if len(c.URLs) == 0 {
		return fmt.Errorf("missing URL\n\nRun %s -h for usage", Name)
	}

	for i, raw := range c.URLs {
		c.URLs[i] = normalizeURL(raw)
	}

	if c.CookieJar != "" && c.Cookie == "" {
		c.Cookie = "@" + c.CookieJar
	}

	if c.Data != "" && !c.MethodSet && !c.UseGET {
		c.Method = "POST"
	}
	if c.UploadFile != "" && !c.MethodSet {
		c.Method = "PUT"
	}
	if c.HeadOnly {
		c.Method = "HEAD"
	}
	c.Method = strings.ToUpper(c.Method)

	if c.Output != "" && c.RemoteName {
		return fmt.Errorf("use either -o or -O, not both")
	}
	if len(c.URLs) > 1 && c.Output != "" {
		return fmt.Errorf("-o with a single filename requires exactly one URL")
	}
	if c.Continue && c.Output == "" && !c.RemoteName {
		return fmt.Errorf("-c requires -o or -O")
	}

	return nil
}

func normalizeURL(raw string) string {
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "https://" + raw
	}
	return raw
}

// Clone returns a copy for fetching another URL in a batch.
func (c *Config) Clone(url string) *Config {
	copy := *c
	copy.URLs = []string{url}
	copy.ResumeOffset = 0
	copy.ResumeFile = ""
	if len(c.Headers) > 0 {
		copy.Headers = append([]string(nil), c.Headers...)
	}
	return &copy
}

// PrintUsage writes CLI help to w.
func PrintUsage(w io.Writer) {
	fmt.Fprintf(w, `%s %s — hybrid curl/wget URL fetcher

Usage:
  %s [options] <url> [url...]
  %s [options] --input-file urls.txt

Run %s -h and see README.md for a full flag guide.

`, Name, Version, Name, Name, Name)
}
