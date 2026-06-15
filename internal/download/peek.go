package download

import (
	"bytes"
	"io"
	"net/http"
)

const binaryPeekSize = 512

// PeekBody reads up to n bytes from the response for inspection and returns
// a new body that replays the peeked bytes followed by the remainder.
func PeekBody(resp *http.Response, n int) ([]byte, error) {
	if n <= 0 {
		return nil, nil
	}
	peek, err := io.ReadAll(io.LimitReader(resp.Body, int64(n)))
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(peek), resp.Body))
	return peek, nil
}

// PeekBodyDefault peeks the default number of bytes used for binary detection.
func PeekBodyDefault(resp *http.Response) ([]byte, error) {
	return PeekBody(resp, binaryPeekSize)
}
