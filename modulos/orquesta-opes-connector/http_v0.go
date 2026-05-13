package orquestaopesconnector

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func (client RESTClientV0) postJSONV0(
	ctx context.Context,
	path string,
	payload any,
	target any,
) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, res.Body)
		return connectorErrorV0{code: ErrOPESHTTPStatusV0}
	}
	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		return connectorErrorV0{code: ErrOPESResponseInvalidV0}
	}
	return nil
}

func trimTrailingSlashV0(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
