package gemini

import "context"

// DefaultClient is the Client is used by the shortcuts Get, and Do.
var DefaultClient = &Client{}

// Get is a wrapper around DefaultClient.Get.
func Get(rawUrl string) (*Response, error) {
	return DefaultClient.Get(rawUrl)
}

// GetContext is a wrapper around DefaultClient.GetContext.
func GetContext(ctx context.Context, rawUrl string) (*Response, error) {
	return DefaultClient.GetContext(ctx, rawUrl)
}

// Do is a wrapper around DefaultClient.Do.
func Do(req *Request) (*Response, error) {
	return DefaultClient.Do(req)
}

// DoContext is a wrapper around DefaultClient.DoContext.
func DoContext(ctx context.Context, req *Request) (*Response, error) {
	return DefaultClient.DoContext(ctx, req)
}
