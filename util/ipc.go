package util


import (
	"fmt"
	"io"
	"os/exec"
)


type ServiceProcess struct {
	cmd     *exec.Cmd
	stdin   chan<- []byte
	stdout  <-chan []byte
	out     []byte
}

func StartServiceProcess(name string, args ...string) (*ServiceProcess, error) {
	var ipipe io.WriteCloser
	var opipe io.ReadCloser
	var in, out chan []byte
	var cmd *exec.Cmd
	var err error

	cmd = exec.Command(name, args...)

	ipipe, err = cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	opipe, err = cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	in = make(chan []byte)
	out = make(chan []byte)

	go writeFromChannel(in, ipipe)
	go readToChannel(opipe, out)

	return &ServiceProcess{
		cmd: cmd,
		stdin: in,
		stdout: out,
		out: nil,
	}, nil
}

func (this *ServiceProcess) Write(p []byte) (int, error) {
	this.stdin <- p

	return len(p), nil
}

func (this *ServiceProcess) Read(p []byte) (int, error) {
	var done int
	var ok bool

	if this.out == nil {
		this.out, ok = <-this.stdout

		if !ok {
			return 0, fmt.Errorf("EOF")
		}
	}

	done = copy(p, this.out)

	if done == len(this.out) {
		this.out = nil
	} else {
		this.out = this.out[done:]
	}

	return done, nil
}

func (this *ServiceProcess) Close() error {
	close(this.stdin)
	return this.cmd.Wait()
}

func writeFromChannel(input <-chan []byte, writer io.WriteCloser) {
	var data, more, acc []byte
	var wchan chan []byte
	var ok bool

	acc = nil
	wchan = make(chan []byte)
	defer close(wchan)

	go forwardFromChannel(wchan, writer)

	outer: for {
		if acc == nil {
			data, ok = <-input
			if !ok {
				break outer
			}
		} else {
			data = acc
			acc = nil
		}

		inner: for {
			select {
			case wchan <- data:
				break inner
			case more, ok = <-input:
				if !ok {
					break outer
				}
				acc = append(acc, more...)
			}
		}
	}
}

func forwardFromChannel(input <-chan []byte, writer io.WriteCloser) {
	var data []byte
	var err error

	defer writer.Close()

	for data = range input {
		_, err = writer.Write(data)
		if err != nil {
			break
		}
	}
}

func readToChannel(reader io.Reader, output chan<- []byte) {
	var data []byte = make([]byte, 256)
	var err error
	var i int

	defer close(output)

	for {
		i, err = reader.Read(data)

		if i > 0 {
			output <- data[:i]

			if i == len(data) {
				data = make([]byte, 2 * len(data))
			} else {
				data = make([]byte, len(data))
			}
		}

		if err != nil {
			break
		}
	}
}
