package gemini

import (
	"bufio"
	"io"
	"path"
	"strings"
)

type wrappedBufferedReader struct {
	buf *bufio.Reader
	rc  io.ReadCloser
}

func (b *wrappedBufferedReader) Read(p []byte) (n int, err error) {
	return b.buf.Read(p)
}

func (b *wrappedBufferedReader) Close() error {
	return b.rc.Close()
}

func (b *wrappedBufferedReader) WriteTo(w io.Writer) (int64, error) {
	return b.buf.WriteTo(w)
}

func pathSegment(path string) (string, string) {
	split := strings.SplitN(path, "/", 2)
	if len(split) != 2 {
		return split[0], ""
	}
	return split[0], split[1]
}

// cleanPath is path.Clean with a few extra steps.
//
// - the path will always start with a slash
// - if the original path ends with a slash, the returned path will as well
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}

	if p[0] != '/' {
		p = "/" + p
	}

	np := path.Clean(p)

	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		// Fast path for common case of p being the string we want:
		if len(p) == len(np)+1 && strings.HasPrefix(p, np) {
			np = p
		} else {
			np += "/"
		}
	}

	return np
}
