package validate

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

// Validator singleton object
var validate *validator.Validate
var once sync.Once

func init() {
	once.Do(func() {
		New()
	})
}

// New initializes singleton object
func New() *validator.Validate {
	if validate != nil {
		return validate
	}
	validate = validator.New()
	return validate
}

// Check validates a structs exposed fields, and automatically validates nested structs, unless otherwise specified.
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise. You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func Check(o interface{}) error {
	e := validate.Struct(o)
	if e != nil {
		for _, ev := range e.(validator.ValidationErrors) {
			return NewFormError(ev.Kind(), ev.Field(), ev.Tag(), ev.Param())
		}
	}
	return nil
}

// Var validates a single variable using tag style validation. eg. var i int validate.Var(i, "gt=1,lt=10")
//
// WARNING: a struct can be passed for validation eg. time.Time is a struct or if you have a custom type and have registered a custom type handler, so must allow it; however unforeseen validations will occur if trying to validate a struct that is meant to be passed to 'validate.Struct'
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise. You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors. validate Array, Slice and maps fields which may contain more than one error
func Var(o interface{}, tag string) error {
	return validate.Var(o, tag)
}
