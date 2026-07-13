package orquestaconfig

import (
	"errors"
	"testing"
)

func TestDecodeAndValidateJSONV0DoesNotValidateOrPublishBeforeDecode(t *testing.T) {
	target := struct {
		Value int `json:"value"`
	}{Value: 99}
	err := DecodeAndValidateJSONV0([]byte(`{"value":0}`), &target, 100, func(value any) error {
		if value.(*struct {
			Value int `json:"value"`
		}).Value == 0 {
			return errors.New("value required")
		}
		return nil
	})
	if err == nil {
		t.Fatal("invalid candidate accepted")
	}
	if target.Value != 99 {
		t.Fatalf("invalid candidate mutated target: %+v", target)
	}
}
