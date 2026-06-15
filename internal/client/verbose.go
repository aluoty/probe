package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aluoty/probe.git/internal/config"
)

// LogResponseSummary prints response status, headers, size, and timing to w.
func LogResponseSummary(w io.Writer, resp *http.Response, elapsed time.Duration, bodyBytes int64) {
	fmt.Fprintf(w, "< HTTP/%d.%d %s\n", resp.ProtoMajor, resp.ProtoMinor, resp.Status)
	for k, vals := range resp.Header {
		for _, v := range vals {
			fmt.Fprintf(w, "< %s: %s\n", k, v)
		}
	}
	if bodyBytes >= 0 {
		fmt.Fprintf(w, "* size: %d bytes\n", bodyBytes)
	} else if cl := resp.ContentLength; cl >= 0 {
		fmt.Fprintf(w, "* size: %d bytes (Content-Length)\n", cl)
	}
	fmt.Fprintf(w, "* time: %s\n", elapsed.Round(time.Millisecond))
}

// LogVerboseRequest is called before the request is sent when -v is set.
func LogVerboseRequest(cfg *config.Config, req *http.Request) {
	if !cfg.Verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "> %s %s\n", req.Method, req.URL)
	for k, vals := range req.Header {
		for _, v := range vals {
			fmt.Fprintf(os.Stderr, "> %s: %s\n", k, v)
		}
	}
	fmt.Fprintln(os.Stderr)
}
