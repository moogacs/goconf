package types

import (
	"bytes"
)

// Status indicates the status of a Rule.
type StatusCode int

const (
	// StatusFailed means someting went wrong. Usually returned when error is also returned.
	StatusFailed StatusCode = iota

	// StatusSatisfied means rule was allready adhered to and no changes had to be made.
	StatusSatisfied

	// StatusEnforced means that the rule did changes to the target
	// The changes was successful.
	StatusEnforced
)

// Response contains the response from a remotely run cmd
type Response struct {
	Stdout     bytes.Buffer
	Stderr     bytes.Buffer
	ExitStatus int
}

// Success checks if an exit code is 0
func (r Response) Success() bool {
	return r.ExitStatus == 0
}

// Success checks the desired status if it's applied or was already there
func (s StatusCode) Success() bool {
	return s == StatusEnforced || s == StatusSatisfied
}
