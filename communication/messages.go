// This file is part of the DIABLO benchmark framework.

// This file lists the messages used during communication of
// the primary and the secondary benchmark clients.

package communication

// Communication Messages
var (
	MsgPrepare  = []byte("\x01") // Initialise the connection
	MsgBc       = []byte("\x02") // 'Blockchain Type'
	MsgWorkload = []byte("\x03") // Workload message
	MsgRun      = []byte("\x04") // Start the benchmark
	MsgResults  = []byte("\x05") // Return the result request
	MsgOk       = []byte("\x99") // Everything is OK
	MsgErr      = []byte("\x98") // There was an error on the client
)
