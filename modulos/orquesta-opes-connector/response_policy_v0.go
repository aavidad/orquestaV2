package orquestaopesconnector

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const defaultOPESResponseMaxBytesV0 int64 = 1 << 20

func decodeOPESJSONResponseV0(response *http.Response, target any) error {
	body, err := readOPESResponseBodyV0(response.Body)
	if err != nil {
		return err
	}
	if !opesContentTypeIsJSONV0(response.Header.Get("Content-Type")) {
		return connectorErrorV0{code: ErrOPESResponseContentTypeV0}
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return connectorErrorV0{code: ErrOPESResponseInvalidV0}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return connectorErrorV0{code: ErrOPESResponseTrailingDataV0}
	}
	return nil
}

func readOPESResponseBodyV0(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, defaultOPESResponseMaxBytesV0+1))
	if err != nil {
		return nil, connectorErrorV0{code: ErrOPESResponseInvalidV0}
	}
	if int64(len(body)) > defaultOPESResponseMaxBytesV0 {
		return nil, connectorErrorV0{code: ErrOPESResponseBodyTooLargeV0}
	}
	return body, nil
}

func discardOPESResponseBodyV0(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, defaultOPESResponseMaxBytesV0))
}

func opesContentTypeIsJSONV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return mediaType == "application/json" || mediaType == "text/plain"
}
