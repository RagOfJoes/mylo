package common

import (
	"reflect"

	goaway "github.com/TwinProduction/go-away"
)

// StringIncludes checks if an array of string includes
// an element
func StringIncludes(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

// Check for profanity given an array of field
// names
func IsProfane(o interface{}, fields []string) bool {
	rf := UnwrapReflectValue(reflect.ValueOf(o))
	for _, f := range fields {
		n := rf.FieldByName(f)
		if !n.IsZero() {
			p := goaway.IsProfane(n.String())
			if p {
				return true
			}
		}
	}
	return false
}

// Continually unwrap until we get the pointer's underlying value
func UnwrapReflectValue(rv reflect.Value) reflect.Value {
	cpy := reflect.Indirect(rv)
	for cpy.Kind() == reflect.Ptr {
		cpy = cpy.Elem()
	}
	return cpy
}
