package main

import (
	"fmt"
	"io"
	"net/http"
)

const commandHTTPResponseMaxBytesV0 int64 = 1 << 20

func readCommandHTTPResponseBodyV0(response *http.Response, context string) ([]byte, error) {
	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("%s_response_missing", context)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, commandHTTPResponseMaxBytesV0+1))
	if err != nil {
		return nil, fmt.Errorf("%s_response_read_error", context)
	}
	if int64(len(body)) > commandHTTPResponseMaxBytesV0 {
		return nil, fmt.Errorf("%s_response_too_large", context)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s_http_%d", context, response.StatusCode)
	}
	return body, nil
}
