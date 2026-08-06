package credentials

import (
	"context"
	"reflect"
	"testing"
)

func TestMaterialSourceContractHasNoByteSliceEgress(t *testing.T) {
	contract := reflect.TypeOf((*MaterialSource)(nil)).Elem()
	method, found := contract.MethodByName("WithSecret")
	if !found || contract.NumMethod() != 1 {
		t.Fatalf("material source contract changed: %v", contract)
	}
	if method.Type.NumIn() != 2 || method.Type.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() ||
		method.Type.In(1) != reflect.TypeOf((func(Secret) error)(nil)) ||
		method.Type.NumOut() != 1 || method.Type.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatalf("unsafe material source signature: %v", method.Type)
	}
	for index := 0; index < method.Type.NumOut(); index++ {
		output := method.Type.Out(index)
		if output.Kind() == reflect.Slice && output.Elem().Kind() == reflect.Uint8 {
			t.Fatalf("material source returns raw bytes: %v", method.Type)
		}
	}
}
