// This file lists the messages used during communication of
// the primary and the secondary benchmark clients.

// Package communication provides the tcp communication between the primary
// and the running secondaries. It is used to transfer commands from the primary
// to the secondary and then run the secondary processes.
package communication

// Communication Messages
var (
	MsgPrepare  = []byte("\x01") // Initialise the connection
	MsgWorkload = []byte("\x02") // Workload message
	MsgRun      = []byte("\x03") // Start the benchmark
	MsgResults  = []byte("\x04") // Return the result request
	MsgFin      = []byte("\x05") // Finish and close the connection.
	MsgOk       = []byte("\x99") // Everything is OK
	MsgErr      = []byte("\x98") // There was an error on the client
)
