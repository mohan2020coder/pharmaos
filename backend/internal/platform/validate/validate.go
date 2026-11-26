package validate

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

var engine = validator.New(validator.WithRequiredStructEnabled())

func Struct(value interface{}) map[string]string {
	err := engine.Struct(value)
	if err == nil {
		return nil
	}
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"_": err.Error()}
	}
	out := make(map[string]string, len(validationErrors))
	for _, fieldError := range validationErrors {
		field := strings.ToLower(fieldError.Field())
		if tag := fieldError.Tag(); tag == "email" {
			out[field] = "must be a valid email address"
		} else if tag == "uuid" {
			out[field] = "must be a valid UUID"
		} else if tag == "min" {
			out[field] = "is too short"
		} else if tag == "max" {
			out[field] = "is too long"
		} else if tag == "oneof" {
			out[field] = "has an invalid value"
		} else {
			out[field] = "is invalid (" + tag + ")"
		}
	}
	return out
}
