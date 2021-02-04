package gemini

import "errors"

var (
	ErrUnknownProtocol = errors.New("unknown protocol")
	ErrUnknownStatus   = errors.New("unknown status")
	ErrAbortHandler    = errors.New("aborted handler")
)
