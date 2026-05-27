package orquestadomainworkhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultDomainWorkHTTPResponseMaxBytesV0 int64 = 1 << 20

func decodeDomainWorkHTTPJSONResponseV0(response *http.Response, target any) error {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, defaultDomainWorkHTTPResponseMaxBytesV0))
		return ErrorV0{Code: fmt.Sprintf("%s_%d", ErrDomainWorkHTTPStatusFailedV0, response.StatusCode)}
	}
	body, err := readDomainWorkHTTPResponseBodyV0(response.Body)
	if err != nil {
		return err
	}
	if !domainWorkHTTPContentTypeIsJSONV0(response.Header.Get("Content-Type")) {
		return ErrorV0{Code: ErrDomainWorkHTTPResponseContentTypeV0}
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return ErrorV0{Code: ErrDomainWorkHTTPResponseDecodeFailedV0}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrorV0{Code: ErrDomainWorkHTTPResponseTrailingDataV0}
	}
	return nil
}

func readDomainWorkHTTPResponseBodyV0(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, defaultDomainWorkHTTPResponseMaxBytesV0+1))
	if err != nil {
		return nil, ErrorV0{Code: ErrDomainWorkHTTPResponseDecodeFailedV0}
	}
	if int64(len(body)) > defaultDomainWorkHTTPResponseMaxBytesV0 {
		return nil, ErrorV0{Code: ErrDomainWorkHTTPResponseBodyTooLargeV0}
	}
	return body, nil
}

func domainWorkHTTPContentTypeIsJSONV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return mediaType == "application/json" || mediaType == "text/plain"
}
