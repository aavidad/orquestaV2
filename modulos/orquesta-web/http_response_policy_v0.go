package orquestaweb

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const webHTTPResponseMaxBytesV0 int64 = 1 << 20

func decodeWebHTTPJSONResponseV0(resp *http.Response, target any) bool {
	body, ok := readWebHTTPResponseBodyV0(resp)
	if !ok {
		return false
	}
	if !webHTTPContentTypeIsJSONV0(resp.Header.Get("Content-Type")) {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return false
	}
	var extra any
	return decoder.Decode(&extra) == io.EOF
}

func readWebHTTPResponseBodyV0(resp *http.Response) ([]byte, bool) {
	if resp == nil || resp.Body == nil {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, webHTTPResponseMaxBytesV0+1))
	if err != nil || int64(len(body)) > webHTTPResponseMaxBytesV0 {
		return nil, false
	}
	return body, true
}

func discardWebHTTPResponseBodyV0(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, webHTTPResponseMaxBytesV0))
}

func webHTTPContentTypeIsJSONV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return mediaType == "application/json" || mediaType == "text/plain"
}
