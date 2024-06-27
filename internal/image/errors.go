package image

import "fmt"

// Error is a basic error type that implements the error interface.
type BasicError struct {
	message string
}

func (e *BasicError) Error() string {
	return e.message
}

// NewError creates a new Error.
func NewError(msg string) *BasicError {
	return &BasicError{message: msg}
}

// SquashError represents an error that occurs during the squashing process.
type SquashError struct {
	BasicError
	code int
}

func (e *SquashError) Error() string {
	return fmt.Sprintf("%s (code: %d)", e.message, e.code)
}

// NewSquashError creates a new SquashError.
func NewSquashError(msg string, code int) *SquashError {
	return &SquashError{BasicError: BasicError{message: msg}, code: code}
}

// SquashUnnecessaryError indicates an error where squashing was unnecessary.
type SquashUnnecessaryError struct {
	SquashError
}

func NewSquashUnnecessaryError(msg string) *SquashUnnecessaryError {
	return &SquashUnnecessaryError{SquashError: *NewSquashError(msg, 2)}
}

// func main() {
// 	// Example of using these errors
// 	err := NewSquashError("Squash failed", 1)
// 	fmt.Println(err.Error())

// 	unnecessaryErr := NewSquashUnnecessaryError("Squashing was unnecessary")
// 	fmt.Println(unnecessaryErr.Error())
// }
