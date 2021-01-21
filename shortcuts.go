package gemini

func Get(rawUrl string) (*Response, error) {
	c := &Client{}

	req, err := NewRequest(rawUrl)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}
