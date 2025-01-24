package util

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// credit: https://github.com/go-playground/validator/issues/559#issuecomment-976459959

type ApiError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func msgForTag(fe validator.FieldError, customField *map[string]string) string {
	// convert to custom field if exist
	field := fe.Field()
	if _, ok := (*customField)[field]; ok {
		field = (*customField)[field]
	}

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%v is required", field)
	case "email":
		return "Invalid email"
	case "numeric":
		return fmt.Sprintf("%v must be numeric", field)
	case "min":
		return fmt.Sprintf("%v must be at least %v characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%v must be at most %v characters", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%v must be greater than or equal to %v", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%v must be less than or equal to %v", field, fe.Param())
	case "eqfield":
		return fmt.Sprintf("%v must be equal to %v", field, fe.Param())
	case "ctime":
		return fmt.Sprintf("%v is not a valid time", field)
	case "cmin":
		return fmt.Sprintf("%v must be at least %v non-whitespace characters", field, fe.Param())
	case "cmax":
		return fmt.Sprintf("%v must be at most %v non-whitespace characters", field, fe.Param())
	case "strNotEmpty":
		return fmt.Sprintf("%v must not be empty or contain only whitespace charaters", field)
	}

	log.Printf("Unknown tag: %v with error: %v", fe.Tag(), fe.Error())
	return fe.Error() // default error
}

/*
Extract error from validator and return as an array of ApiError
Example output:

	 [
	  {
	    "field": "Name",
	    "message": "Name must not be empty or contain only whitespace charaters"
	  }
	]

If customField is provided, it will replace the field name with the custom field name
Example usage: GenerateErrorMessages(err, map[string]string{"name": "CHANGEDFIELDNAME"})
Example output:

	 [
	  {
	    "field": "Name",
	    "message": "CHANGEDFIELDNAME must not be empty or contain only whitespace charaters"
	  }
	]
*/
func GenerateErrorMessages(err error, customField map[string]string) []ApiError {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		out := make([]ApiError, len(ve))
		for i, fe := range ve {
			out[i] = ApiError{fe.Field(), msgForTag(fe, &customField)}
		}
		return out
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []ApiError{
			{
				Field:   "Unknown",
				Message: "Record not found",
			},
		}
	}

	return []ApiError{
		{
			Field:   "Unknown",
			Message: err.Error(),
		},
	}
}

/*
Extract error from validator and return the first error as a string
Usage: GenerateErrorMessagesAsString(err, map[string]string{"email": "Email"})
Example output: "Email is required"
*/
func GenerateErrorMessagesAsString(err error, customField map[string]string) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		if len(ve) > 0 {
			return msgForTag(ve[0], &customField)
		}
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "Record not found"
	}

	return err.Error()
}

// check if string is empty, after trimming spaces
// Usage: `binding:"strNotEmpty"`
func StrNotEmpty(fl validator.FieldLevel) bool {
	// field name. e.g: "email"
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}

	// get the value of the field
	str := field.String()
	str = strings.TrimSpace(str)

	if len(str) == 0 {
		return false
	} else {
		return true
	}
}

// check if string has length of at least the minimum value, after trimming spaces
// Usage: `binding:"cmin=3"`
func CustomMin(fl validator.FieldLevel) bool {
	field := fl.Field()

	// Check if the field is a string
	if field.Kind() != reflect.String {
		return false
	}
	str := field.String()

	minLengthStr := fl.Param()

	// Trim spaces from both sides of the string
	trimmedValue := strings.TrimSpace(str)
	minLengthInt, err := strconv.Atoi(minLengthStr)

	if err != nil {
		return false
	}

	return len(trimmedValue) >= minLengthInt
}

// check if string has length of at most the maximum value, after trimming spaces
// Usage: `binding:"cmax=3"`
func CustomMax(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	str := field.String()

	maxLengthStr := fl.Param()

	// Trim spaces from both sides of the string
	trimmedValue := strings.TrimSpace(str)
	maxLengthInt, err := strconv.Atoi(maxLengthStr)

	if err != nil {
		return false
	}

	return len(trimmedValue) <= maxLengthInt
}
