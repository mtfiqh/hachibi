package hachibi

import (
	"fmt"
	"strings"
)

type Error []error

func (e Error) Error() string {
	errors := make([]string, 0)

	for _, ee := range e {
		errors = append(errors, ee.Error())
	}

	return fmt.Sprintf("[%s]", strings.Join(errors, ", "))
}
