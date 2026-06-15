# probe

A command-line URL fetcher in Go — **curl**-style HTTP requests plus **wget**-style downloads. Standard library only.

## Build

```bash
go build -o probe ./cmd/probe
go install ./cmd/probe
go run ./cmd/probe https://example.com
```

---

## What probe does

| Capability | How |
|------------|-----|
| Send HTTP requests and print responses | Default: `probe URL` prints the body; `-i` adds headers |
| Methods and custom headers | `-X`, `-H` |
| Auto-save binary downloads | Detects binary responses and saves to a file (like wget) |
| GitHub repo as source (no `.git`) | `--github owner/repo[@ref]` downloads ZIP and extracts |
| Debug output | `-v` shows request, response headers, size, and timing |

---

## Usage

```bash
probe [options] <url> [url...]
probe --github owner/repo[@ref]
probe -h
```

---

## Every flag

### HTTP request (curl-like)

| Flag | Purpose |
|------|---------|
| `-X METHOD` | HTTP method. Default `GET`. With `-d` and no `-X`, defaults to `POST`. With `-T`, defaults to `PUT`. |
| `-H "Key: Value"` | Add a request header. Repeat for multiple headers. |
| `-d DATA` | Request body. Use `@file` or `@-` (stdin). |
| `-G` | Append `-d` to the URL as query parameters instead of sending a body. |
| `-T @file` | Upload a file as the request body. |
| `-e URL` | Set the `Referer` header. |
| `-A agent` | Set `User-Agent` (default `probe/0.4.0`). |
| `-u user:pass` | HTTP Basic authentication. |
| `-b cookies` | Send a `Cookie` header (`@file` supported). |
| `--cookie-jar FILE` | Load cookies from FILE; append new `Set-Cookie` values after each response. |
| `-x URL` | HTTP proxy (also uses `HTTP_PROXY` / `HTTPS_PROXY`). |
| `-k` | Skip TLS certificate verification (testing only). |

### Response output

| Flag | Purpose |
|------|---------|
| `-o FILE` | Write the response body to FILE. |
| `-O` | Save using the remote filename (URL or `Content-Disposition`). |
| `-P DIR` | Directory prefix when saving with `-o`, `-O`, or auto binary save. |
| `-i` | Print response headers **and** body to stdout. |
| `-I` | HEAD request — headers only. |
| `-D FILE` | Write response headers to FILE (`-D -` for stdout). |
| `-f` | Exit with code 1 on HTTP 4xx/5xx. |
| `-s` | Silent — hide progress messages (errors hidden unless `-S`). |
| `-S` | Show errors even when `-s` is set. |
| `-v` | **Verbose debug**: request line + headers, then response status + headers + body size + elapsed time. Works with `-s`. |

### Redirects and reliability

| Flag | Purpose |
|------|---------|
| `-L` | Follow HTTP redirects. |
| `--max-redirs N` | Maximum redirects (default 10). |
| `--timeout SEC` | Total request timeout (default 30). |
| `--connect-timeout SEC` | TCP/TLS connect timeout (default 10). |
| `--retry N` | Retry on network failure. |

### Downloads (wget-like)

| Flag | Purpose |
|------|---------|
| *(automatic)* | If the response looks **binary** (image, zip, pdf, `Content-Disposition: attachment`, etc.) and you did not set `-o`/`-O`, probe **auto-saves** using the remote filename. |
| `-c` | Resume a partial download (requires `-o` or `-O`). |
| `-nc` | Do not overwrite an existing file. |
| `--spider` | Check that a URL exists without saving the body. |
| `--input-file FILE` | Read URLs from a file (one per line, `#` comments OK). |

### GitHub

| Flag | Purpose |
|------|---------|
| `--github SPEC` | Download a GitHub repo as a ZIP and extract locally. **No `.git` directory** (GitHub archives never include it). |
| `--github-dir DIR` | Extract into DIR (default: repo name). Combined with `-P` for a parent path. |

**SPEC formats:** `owner/repo`, `owner/repo@ref`, or full `https://github.com/owner/repo`.

**REF** defaults to `main`. Examples: `@master`, `@v1.0.0`, `@go1.22.0`, `@abc1234` (commit).

Ref detection:
- Tags (e.g. `v1.0`, `go1.22.0`) → `refs/tags/…`
- Branch names (e.g. `main`, `dev`) → `refs/heads/…`
- 7+ hex chars → commit SHA

### General

| Flag | Purpose |
|------|---------|
| `-h` | Help. |
| `-V` | Version. |

---

## Examples

### HTTP like curl

```bash
# GET — print HTML/JSON to stdout
probe https://example.com

# Headers + body
probe -i https://example.com

# HEAD only
probe -I https://example.com

# POST JSON (method auto-set to POST)
probe -d '{"ok":true}' -H "Content-Type: application/json" https://httpbin.org/post

# Custom method and headers
probe -X PUT -H "Authorization: Bearer TOKEN" -d @body.json https://api.example.com/item/1

# Follow redirects, fail on errors
probe -L -f -s -o /dev/null https://example.com/redirect
```

### Downloads like wget

```bash
# Explicit save
probe -O https://example.com/files/manual.pdf
probe -o report.pdf https://example.com/report

# Auto-save binary (PNG, ZIP, PDF, …) when no -o/-O given
probe https://example.com/logo.png
# → saves logo.png in the current directory

# Save under a directory
probe -O -P ./downloads/ https://example.com/archive.zip

# Resume + skip existing
probe -O -c -nc -P ./downloads/ https://example.com/large.iso
```

### GitHub repos (no git clone)

```bash
# Latest default branch (main)
probe --github octocat/Hello-World@master

# Specific tag or release
probe --github golang/go@go1.22.0 --github-dir ~/src/go

# Into a parent directory
probe --github owner/repo@dev -P ./vendor --github-dir mylib
# → ./vendor/mylib/
```

Extracted tree has **source files only** — no `.git`, no git history.

### Verbose debugging

```bash
probe -v https://example.com
```

Example stderr output:

```
> GET https://example.com
> User-Agent: probe/0.4.0

< HTTP/1.1 200 OK
< Content-Type: text/html
< ...
* size: 1256 bytes
* time: 83ms
```

Use `-v` with `-I` to debug headers only, or with `--github` to inspect the ZIP download.

---

## Project layout

```
cmd/probe/main.go
internal/config/       flags
internal/client/       HTTP transport, verbose logging
internal/download/     files, binary detection, resume
internal/github/       repo ZIP download and extract
internal/output/       stdout and response handling
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Request, HTTP, or extract error |
| 2 | Invalid usage |

## License

See [LICENSE](LICENSE)
