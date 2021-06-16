package common

import (
	"reflect"
)

// Continually unwrap until we get the pointer's underlying value
func UnwrapReflectValue(rv reflect.Value) reflect.Value {
	cpy := reflect.Indirect(rv)
	for cpy.Kind() == reflect.Ptr {
		cpy = cpy.Elem()
	}
	return cpy
}
