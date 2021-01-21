package gemini

import "io"

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
	return r.Status >= StatusCertificateRequired && r.Status < 70
}
