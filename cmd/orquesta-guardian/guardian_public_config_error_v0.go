package main

import "errors"

func guardianPublicConfigErrorV0(err error) string {
	var parseErr guardianConfigParseErrorV0
	if errors.As(err, &parseErr) && parseErr.Code == guardianConfigExtraArgsV0 {
		return parseErr.Error()
	}
	if code := guardianConfigReasonCodeV0(err); code != "" {
		return code
	}
	return err.Error()
}
