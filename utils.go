package gemini

import (
	"bufio"
	"io"
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
