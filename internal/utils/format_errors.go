package utils

import "strings"

// FormatErrors formats a slice of errors into a single string.
func FormatErrors(errs []error) string {
	var errStrs []string
	for _, err := range errs {
		errStrs = append(errStrs, err.Error())
	}
	return strings.Join(errStrs, "; ")
}
