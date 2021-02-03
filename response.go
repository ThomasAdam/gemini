package gemini

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

// Gemini status codes, as referenced in the spec.
const (
	StatusInput                    int = 10
	StatusSensitiveInput           int = 11
	StatusSuccess                  int = 20
	StatusRedirect                 int = 30
	StatusPermanentRedirect        int = 31
	StatusTemporaryFailure         int = 40
	StatusServerUnavailable        int = 41
	StatusCGIError                 int = 42
	StatusProxyError               int = 43
	StatusSlowDown                 int = 44
	StatusPermanentFailure         int = 50
	StatusNotFound                 int = 51
	StatusGone                     int = 52
	StatusProxyRefusedRequest      int = 53
	StatusBadRequest               int = 59
	StatusCertificateRequired      int = 60
	StatusCertificateNotAuthorized int = 61
	StatusCertificateNotValid      int = 62

	// Sentinel value - this is larger than the current largest valid status
	// code family.
	statusSentinel int = 70
)

// Response represents the response from a Gemini request.
//
// The Client returns Responses from servers once the response status has been
// received. The response body is streamed on demand as the Body field is read.
type Response struct {
	Status int
	Meta   string

	Body io.ReadCloser
}

func NewResponse(status int, meta string, body io.ReadCloser) *Response {
	return &Response{
		Status: status,
		Meta:   meta,
		Body:   body,
	}
}

func NewResponseString(status int, meta string, body string) *Response {
	return NewResponse(status, meta, ioutil.NopCloser(strings.NewReader(body)))
}

func (r *Response) Header() string {
	return fmt.Sprintf("%2d %s", r.Status, r.Meta)
}

// WriteTo implements io.WriterTo for Response.
func (r *Response) WriteTo(w io.Writer) (int64, error) {
	var bytesWritten int64

	n, err := w.Write([]byte(r.Header() + "\r\n"))
	if err != nil {
		return 0, err
	}
	bytesWritten += int64(n)

	n64, err := io.Copy(w, r.Body)
	bytesWritten += n64

	return bytesWritten, err
}

func ReadResponse(conn io.ReadCloser) (*Response, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// This check needs to be here, otherwise TrimSuffix won't be able to
	// guarantee that we're getting valid lines.
	if !strings.HasSuffix(line, "\r\n") {
		return nil, errors.New("malformed status line")
	}

	line = strings.TrimSuffix(line, "\r\n")

	split := strings.SplitN(line, " ", 2)
	if len(split) != 2 {
		return nil, errors.New("invalid response")
	}

	status, err := strconv.Atoi(split[0])
	if err != nil {
		return nil, err
	}

	return &Response{
		Status: status,
		Meta:   split[1],
		Body: &wrappedBufferedReader{
			buf: reader,
			rc:  conn,
		},
	}, nil
}

// IsInput is a convenience method for determining if this response status
// represents a input request.
func (r *Response) IsInput() bool {
	return r.Status >= StatusInput && r.Status < StatusSuccess
}

// IsSuccess is a convenience method for determining if this response status
// represents a success.
func (r *Response) IsSuccess() bool {
	return r.Status >= StatusSuccess && r.Status < StatusRedirect
}

// IsRedirect is a convenience method for determining if this response status
// represents a redirect.
func (r *Response) IsRedirect() bool {
	return r.Status >= StatusRedirect && r.Status < StatusTemporaryFailure
}

// IsTemporaryFailure is a convenience method for determining if this response
// status represents a temporary failure.
func (r *Response) IsTemporaryFailure() bool {
	return r.Status >= StatusTemporaryFailure && r.Status < StatusPermanentFailure
}

// IsPermanentFailure is a convenience method for determining if this response
// status represents a permanent failure.
func (r *Response) IsPermanentFailure() bool {
	return r.Status >= StatusPermanentFailure && r.Status < StatusCertificateRequired
}

// IsCertificateRequired is a convenience method for determining if this
// response status represents a client certificate failure.
func (r *Response) IsCertificateRequired() bool {
	return r.Status >= StatusCertificateRequired && r.Status < statusSentinel
}
