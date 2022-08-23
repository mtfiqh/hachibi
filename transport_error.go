package hachibi

import "strings"

type Error struct {
	errors []error
}

func (e *Error) Error() string {
	errors := make([]string, 0)

	for _, ee := range e.errors {
		errors = append(errors, ee.Error())
	}

	return strings.Join(errors, ", ")
}
