package orquestaconfig

import (
	"errors"
	"fmt"
	"reflect"
)

type CandidateValidatorV0 func(candidate any) error

func ValidateCandidateV0(candidate any, validators ...CandidateValidatorV0) error {
	for _, validator := range validators {
		if validator == nil {
			return errors.New("orquesta-config: nil candidate validator")
		}
		if err := validator(candidate); err != nil {
			return fmt.Errorf("orquesta-config: candidate invalid: %w", err)
		}
	}
	return nil
}

func DecodeAndValidateJSONV0(data []byte, target any, maxBytes int, validators ...CandidateValidatorV0) error {
	if target == nil {
		return ErrInvalidTargetV0
	}
	// DecodeStrictBoundedJSONV0 deliberately does not publish an invalid
	// candidate. Validation needs the same guarantee, so it works through a
	// temporary value and copies it into target only after every validator.
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Pointer || targetValue.IsNil() {
		return ErrInvalidTargetV0
	}
	candidate := reflect.New(targetValue.Elem().Type())
	if err := DecodeStrictBoundedJSONV0(data, candidate.Interface(), maxBytes); err != nil {
		return err
	}
	if err := ValidateCandidateV0(candidate.Interface(), validators...); err != nil {
		return err
	}
	targetValue.Elem().Set(candidate.Elem())
	return nil
}
