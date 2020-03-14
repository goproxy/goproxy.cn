package base

import "net/http"

// HTTPDo sends the req via the client and returns an `http.Response`. It will
// automatically retry when it encounters 502, 503, and 504 (up to 5 times).
//
// If the client is nil, the `http.DefaultClient` is used.
func HTTPDo(client *http.Client, req *http.Request) (*http.Response, error) {
	if client == nil {
		client = http.DefaultClient
	}

	nots := 0

Do:
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		if nots < 5 {
			nots++
			goto Do
		}

		return nil, err
	}

	switch res.StatusCode {
	case http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		if nots < 5 {
			nots++
			res.Body.Close()
			goto Do
		}
	}

	return res, nil
}
