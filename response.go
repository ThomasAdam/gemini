package gemini

import "io"

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

type Response struct {
	Status int
	Meta   string

	Body io.ReadCloser
}

func (r *Response) IsSuccess() bool {
	return r.Status >= StatusSuccess && r.Status < StatusRedirect
}

func (r *Response) IsRedirect() bool {
	return r.Status >= StatusRedirect && r.Status < StatusTemporaryFailure
}
