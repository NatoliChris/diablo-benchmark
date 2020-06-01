package communication

import (
	"fmt"
)

// When the benchmark fails to send or receive from
// a client.
type ClientCommError struct {
	ClientInfo string // Client Information
	Err        error  // The error message.
}

// When the benchmark receives a client error
// message.
type ClientErrorReply struct {
	Info string // Information about the client
	Err  error  // The actual error we want to send
}

// Error message for the client communication error
func (e *ClientCommError) Error() string {
	return fmt.Sprintf("failed to send to %s: %s", e.ClientInfo, e.Err.Error())
}

// Error message if we received an error reply from a client
func (e *ClientErrorReply) Error() string {
	return fmt.Sprintf("[%s]: %s", e.Info, e.Err.Error())
}
