# probe

A command-line URL fetcher in Go — **curl**-style HTTP control plus **wget**-style downloading. Standard library only.

**probe covers the everyday HTTP workflow** (fetch, POST, download, resume, spider, auth, cookies, proxy). It does **not** replace every curl/wget feature — see [What probe does not do](#what-probe-does-not-do) below.

## Build

```bash
go build -o probe ./cmd/probe
go install ./cmd/probe
go run ./cmd/probe https://example.com
```

## Basic usage

```bash
probe [options] <url> [url...]
probe [options] --input-file urls.txt
probe -h          # short help
probe -V          # version
```

---

## Every flag explained

### Request control (curl-like)

| Flag | Name | What it does |
|------|------|--------------|
| `-X METHOD` | request | HTTP method. Default `GET`. Common values: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`. |
| `-H "Key: Value"` | header | Add a request header. Repeat for multiple headers. Example: `-H "Accept: application/json"`. |
| `-d DATA` | data | Request body. Plain string, or `@file` to read from a file, or `@-` for stdin. **If you send `-d` without `-X`, probe automatically uses `POST`** (curl behavior). |
| `-G` | get | Send `-d` data as a **URL query string** instead of a body. Implies GET. Example: `-G -d "q=hello"` → `?q=hello`. |
| `-T FILE` | upload | Upload a file as the request body (`@path` or `@-`). **Defaults to `PUT`** unless you set `-X`. |
| `-e URL` | referer | Set the `Referer` request header. |
| `-A AGENT` | user-agent | Set the `User-Agent` header. Default: `probe/0.3.0`. |
| `-u user:pass` | user | HTTP Basic authentication. |
| `-b COOKIES` | cookie | Send a `Cookie` header. Plain string (`session=abc`) or `@file` to read from a file. |
| `--cookie-jar FILE` | cookie jar | **Read** cookies from FILE on request (if `-b` is not set) and **append** `Set-Cookie` lines from the response after each fetch. Simple line-based format, not full Netscape jar parsing. |
| `-x URL` | proxy | Route the request through a proxy, e.g. `http://127.0.0.1:8080`. Also respects `HTTP_PROXY` / `HTTPS_PROXY` env vars when `-x` is not set. |
| `-k` | insecure | Skip TLS certificate verification (like curl `-k` / wget `--no-check-certificate`). **Use only for testing.** |

### Response & output

| Flag | Name | What it does |
|------|------|--------------|
| `-o FILE` | output | Write the response **body** to FILE instead of stdout. |
| `-O` | remote-name | Save the body using the remote filename (from the URL path or `Content-Disposition` header). |
| `-P DIR` | directory-prefix | Prepend a directory when saving with `-o` or `-O`. Creates the directory if needed. Example: `-O -P ./downloads/`. |
| `-i` | include | Print response **headers and body** to stdout (headers first, then a blank line, then body). |
| `-I` | head | Fetch **headers only** (sends a HEAD request). Prints status line + headers, or just status without `-i`. |
| `-D FILE` | dump-header | Write response headers to FILE. Use `-D -` for stdout. Body still goes to stdout or `-o`/`-O` as usual. |
| `-f` | fail | Exit with code **1** on HTTP **4xx/5xx** instead of treating it as success. |
| `-s` | silent | Suppress progress messages on stderr. Errors are hidden too unless `-S` is set. |
| `-S` | show-error | Show errors even when `-s` is active (curl `-S`). |
| `-v` | verbose | Print request line, request headers, and redirect targets to stderr. |

### Redirects & timing

| Flag | Name | What it does |
|------|------|--------------|
| `-L` | location | Follow HTTP redirects (301, 302, 307, …). |
| `--max-redirs N` | max redirects | Stop after N redirects. Default **10**. |
| `--timeout SEC` | timeout | Total time limit for the entire request (connect + transfer). Default **30** seconds. |
| `--connect-timeout SEC` | connect timeout | Time limit to establish a TCP/TLS connection. Default **10** seconds. |
| `--retry N` | retry | Retry N times on network errors, with a short backoff between attempts. |

### Download behavior (wget-like)

| Flag | Name | What it does |
|------|------|--------------|
| `-c` | continue | **Resume** a partial download. Requires `-o` or `-O`. Sends a `Range` header and appends to the existing file. |
| `-nc` | no-clobber | **Do not overwrite** an existing output file. Skips the download if the target file already exists. |
| `--spider` | spider | Check whether the URL is reachable **without saving the body** (like `wget --spider`). Prints result to stderr. |
| `--input-file FILE` | input file | Read additional URLs from FILE, one per line. Lines starting with `#` are ignored. Can combine with URLs on the command line. |

### General

| Flag | Name | What it does |
|------|------|--------------|
| `-h` | help | Show short usage text. |
| `-V` | version | Print version and exit. |

---

## Usage examples

### Fetch & inspect

```bash
# Page to stdout
probe https://example.com

# Headers only
probe -I https://example.com

# Headers + body together
probe -i https://example.com

# Save headers to a file
probe -D headers.txt -o body.html https://example.com
```

### POST, PUT, and query strings

```bash
# POST JSON (method auto-set to POST)
probe -d '{"name":"probe"}' -H "Content-Type: application/json" https://api.example.com/items

# POST from a file
probe -d @payload.json -H "Content-Type: application/json" https://api.example.com/items

# GET with query parameters
probe -G -d "search=golang&page=1" https://api.example.com/search

# Upload a file
probe -T @photo.jpg -H "Content-Type: image/jpeg" https://api.example.com/upload
```

### Download

```bash
# Save as remote filename
probe -O https://example.com/files/manual.pdf

# Save into a directory
probe -O -P ./downloads/ https://example.com/archive.zip

# Named output file
probe -o report.pdf https://example.com/report

# Resume an interrupted download
probe -O -c -P ./downloads/ https://example.com/large.iso

# Skip if file already exists
probe -O -nc -P ./downloads/ https://example.com/file.zip
```

### Batch URLs

```bash
# Multiple URLs with remote filenames
probe -O https://example.com/a.html https://example.com/b.html

# URLs from a file
probe -O --input-file urls.txt
```

`urls.txt`:
```
https://example.com/page1
https://example.com/page2
# comments allowed
```

### Auth, cookies, proxy

```bash
# Basic auth
probe -u admin:secret https://api.example.com/private

# Send cookies
probe -b "session=abc123" https://app.example.com/dashboard

# Persist cookies across requests
probe -b @cookies.txt --cookie-jar cookies.txt https://app.example.com/login

# Through a proxy
probe -x http://127.0.0.1:8080 https://example.com

# Ignore bad TLS certs (testing only)
probe -k https://self-signed.local/
```

### Redirects, retries, errors

```bash
# Follow redirects
probe -L -I https://short.link/abc

# Fail on 404/500 (useful in scripts)
probe -f -s -o /dev/null https://example.com/must-exist

# Retry on network failure
probe --retry 3 --timeout 15 https://flaky.example.com/data

# Spider check (exit 0 = reachable)
probe --spider https://example.com
echo $?
```

### Scripting

```bash
# Silent fetch, fail on HTTP errors, only body to stdout
probe -sS -f https://example.com/api/data

# Verbose debug of redirects
probe -v -L https://example.com/redirect-chain
```

---

## curl / wget compatibility

### Covered well (daily use)

| Feature | curl | wget | probe |
|---------|:----:|:----:|:-----:|
| GET/POST/PUT/DELETE | ✓ | partial | ✓ |
| Custom headers | ✓ | partial | ✓ |
| Request body / upload | ✓ | | ✓ |
| Query string from data (`-G`) | ✓ | | ✓ |
| Basic auth | ✓ | ✓ | ✓ |
| Cookies send/save | ✓ | ✓ | ✓ (simple jar) |
| Follow redirects | ✓ | ✓ | ✓ |
| Save to file / remote name | ✓ | ✓ | ✓ |
| Resume download | ✓ | ✓ | ✓ |
| Spider / HEAD check | ✓ | ✓ | ✓ |
| Proxy | ✓ | ✓ | ✓ |
| Skip TLS verify | ✓ | ✓ | ✓ |
| Retries & timeouts | ✓ | partial | ✓ |
| Multiple URLs | ✓ | ✓ | ✓ |
| Directory prefix | | ✓ | ✓ |
| No-clobber | | ✓ | ✓ |
| Silent / verbose | ✓ | ✓ | ✓ |

### What probe does **not** do

These are outside probe's scope (use curl or wget instead):

| Feature | Tool |
|---------|------|
| Recursive / mirror website crawl (`wget -r`, `wget -m`) | wget |
| FTP / SFTP / SCP / file:// transfers | curl, wget |
| Multipart form upload (`curl -F`) | curl |
| OAuth, NTLM, Kerberos, client certificates | curl |
| `curl -w` write-out format strings | curl |
| Rate limiting / bandwidth throttle | wget |
| HTML link rewriting / page requisites | wget |
| Full Netscape cookie jar semantics | curl |
| SOCKS proxy (use `ALL_PROXY=socks5://…` with curl) | curl |
| HTTP/2 or HTTP/3 manual control | curl |
| Parallel downloads | wget, aria2 |

---

## Project layout

```
cmd/probe/main.go       entry point, multi-URL loop
internal/config/        flags and validation
internal/client/        HTTP transport, body, cookies
internal/download/      files, resume, clobber
internal/output/        stdout, headers, spider
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
| 1 | Network error, HTTP error (with `-f`), or failed fetch in batch |
| 2 | Invalid usage |

## License

See [LICENSE](LICENSE)
