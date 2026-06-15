# probe

A small command-line URL fetcher in Go — **curl**-style HTTP control plus **wget**-style downloads. Standard library only, no dependencies.

## Build

```bash
go build -o probe ./cmd/probe
```

Install onto your `PATH`:

```bash
go install ./cmd/probe
```

Run without building:

```bash
go run ./cmd/probe https://example.com
```

## Usage

```bash
probe [options] <url>
probe -h          # full flag list
probe -V          # version
```

### Quick examples

```bash
probe https://example.com
probe -I https://example.com
probe -O https://example.com/index.html
probe -O -c https://example.com/large.zip          # resume download
probe --spider https://example.com                 # existence check
probe -X POST -d '{"ok":true}' -H "Content-Type: application/json" https://httpbin.org/post
```

### Flags

| Flag | Description |
|------|-------------|
| `-X` | HTTP method (default `GET`) |
| `-H` | Request header (`Key: Value`), repeatable |
| `-d` | Request body; `@file` or `@-` for stdin |
| `-o` | Write body to file |
| `-O` | Save using remote filename |
| `-i` | Include response headers in output |
| `-I` | Headers only (HEAD) |
| `-L` | Follow redirects (max 10) |
| `-A` | User-Agent |
| `-u` | Basic auth (`user:password`) |
| `-f` | Exit with error on HTTP 4xx/5xx |
| `-s` | Silent |
| `-v` | Verbose |
| `-c` | Resume partial download (requires `-o` or `-O`) |
| `--spider` | Check URL without downloading |
| `--retry` | Retry on network failure |
| `--timeout` | Timeout in seconds (default 30) |

## Layout

```
cmd/probe/main.go          entry point
internal/config/           flags and help
internal/client/           HTTP requests
internal/download/         file output and resume
internal/output/           stdout and spider mode
```

## Development

```bash
go build -o probe ./cmd/probe
go test ./...
go vet ./...
go fmt ./...
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Request or HTTP error |
| 2 | Bad usage |

## License

MIT
