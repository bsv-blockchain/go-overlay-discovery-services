package shared

import (
	"reflect"
)

// ValidateStringPtrField validates that a *string field is actually a string pointer.
// Returns the provided error if the value is non-nil but not a *string.
func ValidateStringPtrField(field *string, fieldErr error) error {
	if field != nil {
		if reflect.TypeOf(field).Kind() != reflect.Ptr ||
			reflect.TypeOf(field).Elem().Kind() != reflect.String {
			return fieldErr
		}
	}
	return nil
}
