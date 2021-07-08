package common

import (
	"reflect"

	"github.com/RagOfJoes/idp/session"
	"github.com/gin-gonic/gin"
)

// Continually unwrap until we get the pointer's underlying value
func UnwrapReflectValue(rv reflect.Value) reflect.Value {
	cpy := reflect.Indirect(rv)
	for cpy.Kind() == reflect.Ptr {
		cpy = cpy.Elem()
	}
	return cpy
}

// IsAuthenticated checks context for identity
func IsAuthenticated(ctx *gin.Context) bool {
	if id, ok := ctx.Value("auth_session").(*session.AuthSession); ok && id != nil {
		return true
	}
	return false
}
