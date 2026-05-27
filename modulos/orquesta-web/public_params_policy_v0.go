package orquestaweb

import (
	"errors"
	"net/http"
	"net/url"
)

const (
	webPublicQueryMaxBytesV0        = 8 << 10
	webPublicParamMaxKeysV0         = 80
	webPublicParamMaxValuesPerKeyV0 = 32
	webPublicParamMaxKeyBytesV0     = 96
	webPublicParamMaxValueBytesV0   = 4 << 10
	webPublicParamMaxTotalBytesV0   = 64 << 10
)

var errWebPublicQueryTooLargeV0 = errors.New("public_query_too_large")
var errWebPublicParamTooLargeV0 = errors.New("public_parameter_too_large")
var errWebPublicParamRepeatedV0 = errors.New("public_parameter_repeated")
var errWebPublicParamTooManyV0 = errors.New("public_parameter_too_many")

func validateWebPublicQueryV0(r *http.Request) error {
	if r == nil || r.URL == nil || r.URL.RawQuery == "" {
		return nil
	}
	if len(r.URL.RawQuery) > webPublicQueryMaxBytesV0 {
		return errWebPublicQueryTooLargeV0
	}
	return validateWebPublicValuesV0(r.URL.Query())
}

func webPublicQueryValuesV0(r *http.Request) url.Values {
	if err := validateWebPublicQueryV0(r); err != nil || r == nil || r.URL == nil {
		return url.Values{}
	}
	return r.URL.Query()
}

func validateWebPublicValuesV0(values url.Values) error {
	if len(values) > webPublicParamMaxKeysV0 {
		return errWebPublicParamTooManyV0
	}
	total := 0
	for key, items := range values {
		if len(key) > webPublicParamMaxKeyBytesV0 {
			return errWebPublicParamTooLargeV0
		}
		if len(items) > webPublicParamMaxValuesPerKeyV0 {
			return errWebPublicParamRepeatedV0
		}
		total += len(key)
		for _, value := range items {
			if len(value) > webPublicParamMaxValueBytesV0 {
				return errWebPublicParamTooLargeV0
			}
			total += len(value)
			if total > webPublicParamMaxTotalBytesV0 {
				return errWebPublicParamTooLargeV0
			}
		}
	}
	return nil
}
