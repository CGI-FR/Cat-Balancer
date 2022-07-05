// Copyright (C) 2021 CGI France
//
// This file is part of Cat Balancer.
//
// Cat Balancer is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Cat Balancer is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Cat Balancer.  If not, see <http://www.gnu.org/licenses/>.

package catp

import (
	"io"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// Pipe stream data from input to output.
type Pipe interface {
	Configure(input io.Reader, output io.Writer, doneHandler func(), errorHandler func(error)) io.Reader
	Start()
}

type SimplePipe struct {
	doneHandler  func()
	errorHandler func(error)
	input        io.Reader
	output       io.Writer
}

func (p *SimplePipe) Configure(
	input io.Reader, output io.Writer,
	doneHandler func(),
	errorHandler func(error)) io.Reader {
	p.doneHandler = doneHandler
	p.errorHandler = errorHandler
	p.input = input
	p.output = output

	return input
}

// Start a goroutine to copy data from input to output.
func (p *SimplePipe) Start() {
	go func() {
		_, err := io.Copy(p.output, p.input)
		if err != nil {
			p.errorHandler(err)
		} else {
			p.doneHandler()
		}
	}()
}

// ProcessConfiguration is the process execute by catp.
type ProcessConfiguration struct {
	Command []string
	Stderr  io.Writer
}

func NewProcessPipe(configuration ProcessConfiguration) *ProcessPipe {
	// nolint: exhaustivestruct
	return &ProcessPipe{SimplePipe{}, configuration}
}

func (p *ProcessPipe) Start() {
	go func() {
		// nolint: gosec
		cmd := exec.Command(p.Command[0], p.Command[1:]...)
		cmd.Stdin = p.input
		cmd.Stdout = p.output

		if p.ProcessConfiguration.Stderr != nil {
			cmd.Stderr = p.ProcessConfiguration.Stderr
		}

		err := cmd.Start()
		if err != nil {
			p.errorHandler(err)

			return
		}

		err = cmd.Wait()
		if err != nil {
			p.errorHandler(err)

			return
		}

		p.doneHandler()
	}()
}

type ProcessPipe struct {
	SimplePipe
	ProcessConfiguration
}

// New build a new catp chain.
func New() Chain {
	// nolint: exhaustivestruct
	return Chain{}
}

type Chain struct {
	pipes []Pipe
}

func (c *Chain) Add(pipe Pipe) {
	c.pipes = append(c.pipes, pipe)
}

func (c *Chain) Run(input io.Reader, output io.Writer) (result error) {
	errChan := make(chan error, 1)
	doneChan := make(chan struct{}, 1)

	doneHandler := func() {
		doneChan <- struct{}{}
	}

	errorHandler := func(err error) {
		errChan <- err
	}

	for _, pipe := range c.pipes {
		input = pipe.Configure(input, output, doneHandler, errorHandler)
	}

	for _, pipe := range c.pipes {
		pipe.Start()
	}

	for range c.pipes {
		select {
		case err := <-errChan:
			log.Error().Err(err).Msg("Pipe Failed")

			return err

		case <-doneChan:
		}
	}

	return nil
}

// Start create  a catp chain and start it.
func Start(command []string, streamIn io.Reader, streamOut io.Writer, captureStderr io.Writer) error {
	chain := New()

	if len(command) > 0 {
		processConfiguation := ProcessConfiguration{
			Command: command,
			Stderr:  captureStderr,
		}
		chain.Add(NewProcessPipe(processConfiguation))
	} else {
		// nolint: exhaustivestruct
		chain.Add(&SimplePipe{})
	}

	err := chain.Run(streamIn, streamOut)
	if err != nil {
		return err
	}

	return nil
}
