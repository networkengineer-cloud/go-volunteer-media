package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// formatValidationError converts a validator.ValidationErrors into a user-friendly
// message. It uses JSON field names and maps tag names to readable descriptions.
// If err is not a validator.ValidationErrors, it returns the error string as-is.
func formatValidationError(err error) string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err.Error()
	}
	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		msgs = append(msgs, fieldErrMsg(fe))
	}
	return strings.Join(msgs, "; ")
}

func fieldErrMsg(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	case "alphanum":
		return fmt.Sprintf("%s may only contain letters and numbers", field)
	case "usernamechars":
		return fmt.Sprintf("%s may only contain letters, numbers, underscores, dots, and dashes", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
