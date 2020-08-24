package communication

import (
	"fmt"
)

// When the benchmark fails to send or receive from
// a secondary.
type SecondaryCommError struct {
	SecondaryInfo string // Secondary Information
	Err           error  // The error message.
}

// When the benchmark receives a secondary error
// message.
type SecondaryErrorReply struct {
	Info string // Information about the secondary
	Err  error  // The actual error we want to send
}

// Error message for the secondary communication error
func (e *SecondaryCommError) Error() string {
	return fmt.Sprintf("failed to send to %s: %s", e.SecondaryInfo, e.Err.Error())
}

// Error message if we received an error reply from a secondary
func (e *SecondaryErrorReply) Error() string {
	return fmt.Sprintf("[%s]: %s", e.Info, e.Err.Error())
}
