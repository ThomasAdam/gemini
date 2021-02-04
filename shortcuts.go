package gemini

// DefaultClient is the Client is used by the shortcuts Get, and Do.
var DefaultClient = &Client{}

// Get is a wrapper around DefaultClient.Do which also parses the given URL.
func Get(rawUrl string) (*Response, error) {
	req, err := NewRequest(rawUrl)
	if err != nil {
		return nil, err
	}

	return DefaultClient.Do(req)
}

// Do is a wrapper around DefaultClient.Do.
func Do(req *Request) (*Response, error) {
	return DefaultClient.Do(req)
}
