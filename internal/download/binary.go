package download

import (
	"net/http"
	"strings"
)

// ShouldSaveBinary reports whether a response body should be saved to disk
// instead of printed to stdout (wget-style for binary content).
func ShouldSaveBinary(resp *http.Response, peek []byte) bool {
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		media := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
		if isTextMediaType(media) {
			return false
		}
		if isBinaryMediaType(media) {
			return true
		}
	}

	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		lower := strings.ToLower(cd)
		if strings.Contains(lower, "attachment") || strings.Contains(lower, "filename=") {
			return true
		}
	}

	return looksBinary(peek)
}

func isTextMediaType(media string) bool {
	switch {
	case media == "application/json",
		media == "application/xml",
		media == "application/javascript",
		media == "application/x-javascript",
		media == "application/ld+json",
		media == "application/xhtml+xml":
		return true
	case strings.HasPrefix(media, "text/"):
		return true
	case strings.HasPrefix(media, "application/vnd.") && (strings.Contains(media, "json") || strings.Contains(media, "xml")):
		return true
	}
	return false
}

func isBinaryMediaType(media string) bool {
	switch {
	case strings.HasPrefix(media, "image/"),
		strings.HasPrefix(media, "audio/"),
		strings.HasPrefix(media, "video/"),
		strings.HasPrefix(media, "font/"):
		return true
	case media == "application/octet-stream",
		media == "application/pdf",
		media == "application/zip",
		media == "application/gzip",
		media == "application/x-gzip",
		media == "application/x-tar",
		media == "application/x-7z-compressed",
		media == "application/x-bzip2",
		media == "application/msword",
		media == "application/vnd.ms-excel",
		media == "application/vnd.ms-powerpoint":
		return true
	case strings.HasPrefix(media, "application/vnd.") && !strings.Contains(media, "json") && !strings.Contains(media, "xml"):
		return true
	}
	return false
}

func looksBinary(peek []byte) bool {
	if len(peek) == 0 {
		return false
	}
	for _, b := range peek {
		if b == 0 {
			return true
		}
	}
	nonPrint := 0
	for _, b := range peek {
		if b < 0x09 || (b > 0x0d && b < 0x20) {
			nonPrint++
		}
	}
	return nonPrint*4 > len(peek)*3
}
