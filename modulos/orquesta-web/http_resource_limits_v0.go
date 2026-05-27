package orquestaweb

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

const (
	webControlJSONMaxBytesV0 int64 = 256 << 10
	webControlFormMaxBytesV0 int64 = 256 << 10
)

func decodeWebControlJSONV0(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, webControlJSONMaxBytesV0))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request_body_trailing_data")
	}
	return nil
}

func parseWebControlFormV0(w http.ResponseWriter, r *http.Request) error {
	if err := validateWebPublicQueryV0(r); err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, webControlFormMaxBytesV0)
	if err := r.ParseForm(); err != nil {
		return err
	}
	return validateWebPublicValuesV0(r.Form)
}

func webControlContentTypeAllowsJSONV0(value string) bool {
	mediaType := webControlMediaTypeV0(value)
	return mediaType == "" || mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func webControlContentTypeAllowsFormV0(value string) bool {
	mediaType := webControlMediaTypeV0(value)
	return mediaType == "application/x-www-form-urlencoded" || mediaType == "multipart/form-data"
}

func webControlMediaTypeV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(value, ";")[0])
	}
	return strings.ToLower(mediaType)
}
