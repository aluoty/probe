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
	Version   = "0.2.0"
	UserAgent = Name + "/" + Version
)

// Config holds all CLI options for probe.
type Config struct {
	URL            string
	Method         string
	Headers        []string
	Data           string
	Output         string
	RemoteName     bool
	IncludeHeaders bool
	HeadOnly       bool
	FollowRedirect bool
	Silent         bool
	Verbose        bool
	UserAgent      string
	BasicAuth      string
	Timeout        time.Duration
	Continue       bool
	Spider         bool
	Retry          int
	FailOnError    bool
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
	cfg := &Config{UserAgent: UserAgent}

	fs := flag.NewFlagSet(Name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&cfg.Method, "X", "GET", "HTTP method")
	var headers flagArray
	fs.Var(&headers, "H", "Request header (Key: Value)")
	fs.StringVar(&cfg.Data, "d", "", "Request body (@file or @- for stdin)")
	fs.StringVar(&cfg.Output, "o", "", "Write body to FILE")
	fs.BoolVar(&cfg.RemoteName, "O", false, "Save using remote filename")
	fs.BoolVar(&cfg.IncludeHeaders, "i", false, "Include response headers in output")
	fs.BoolVar(&cfg.HeadOnly, "I", false, "Fetch headers only")
	fs.BoolVar(&cfg.FollowRedirect, "L", false, "Follow redirects")
	fs.BoolVar(&cfg.Silent, "s", false, "Silent mode")
	fs.BoolVar(&cfg.Verbose, "v", false, "Verbose mode")
	fs.StringVar(&cfg.UserAgent, "A", UserAgent, "User-Agent")
	fs.StringVar(&cfg.BasicAuth, "u", "", "Basic auth (user:password)")

	var timeoutSec float64
	fs.Float64Var(&timeoutSec, "timeout", 30, "Timeout in seconds")

	fs.BoolVar(&cfg.Continue, "c", false, "Resume partial download")
	fs.BoolVar(&cfg.Spider, "spider", false, "Check URL without downloading")
	fs.IntVar(&cfg.Retry, "retry", 0, "Retries on failure")
	fs.BoolVar(&cfg.FailOnError, "f", false, "Exit with error on HTTP 4xx/5xx")
	fs.BoolVar(&cfg.ShowHelp, "h", false, "Show help")
	fs.BoolVar(&cfg.ShowVersion, "V", false, "Show version")

	fs.Usage = func() { PrintUsage(fs.Output()) }

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.Headers = headers
	cfg.Timeout = time.Duration(timeoutSec * float64(time.Second))

	if rest := fs.Args(); len(rest) > 0 {
		cfg.URL = rest[0]
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.ShowHelp || c.ShowVersion {
		return nil
	}
	if c.URL == "" {
		return fmt.Errorf("missing URL\n\nRun %s -h for usage", Name)
	}
	if !strings.HasPrefix(c.URL, "http://") && !strings.HasPrefix(c.URL, "https://") {
		c.URL = "https://" + c.URL
	}
	if c.HeadOnly {
		c.Method = "HEAD"
	}
	c.Method = strings.ToUpper(c.Method)
	return nil
}

// PrintUsage writes CLI help to w.
func PrintUsage(w io.Writer) {
	fmt.Fprintf(w, `%s %s — hybrid curl/wget URL fetcher

Usage:
  %s [options] <url>

Options:
  -X METHOD       HTTP method (default GET)
  -H "K: V"       Request header (repeatable)
  -d DATA         Request body (@file or @- for stdin)
  -o FILE         Write body to file
  -O              Save using remote filename
  -i              Include response headers in output
  -I              Headers only (HEAD)
  -L              Follow redirects
  -A AGENT        User-Agent (default %s)
  -u USER:PASS    Basic authentication
  -f              Fail on HTTP 4xx/5xx
  -s              Silent
  -v              Verbose
  -c              Resume partial download (needs -o or -O)
  --spider        Check URL without downloading
  --retry N       Retry count (default 0)
  --timeout SEC   Timeout in seconds (default 30)
  -h              Help
  -V              Version

Examples:
  %s https://example.com
  %s -I https://example.com
  %s -X POST -d '{"x":1}' -H "Content-Type: application/json" https://httpbin.org/post
  %s -O -c https://example.com/file.zip

`, Name, Version, Name, UserAgent, Name, Name, Name, Name)
}
