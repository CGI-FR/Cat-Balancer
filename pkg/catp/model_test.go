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

package catp_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/cgi-fr/cat-balancer/pkg/catp"
	"github.com/stretchr/testify/assert"
)

func TestSimplePipe(t *testing.T) {
	t.Parallel()

	simple := catp.SimplePipe{}

	reader := strings.NewReader("hello\n")
	writer := &strings.Builder{}

	wg := sync.WaitGroup{}
	wg.Add(1)

	doneHandler := func() {
		wg.Done()
	}

	errorHandler := func(error) {
		assert.Fail(t, "should not be call")
	}

	simple.Configure(reader, writer, doneHandler, errorHandler)

	simple.Start()

	wg.Wait()
}

func TestChainSimplePipe(t *testing.T) {
	t.Parallel()

	chain := catp.New()

	chain.Add(&catp.SimplePipe{})

	reader := strings.NewReader("hello\n")
	writer := &strings.Builder{}

	err := chain.Run(reader, writer)
	assert.Equal(t, "hello\n", writer.String())
	assert.Nil(t, err)
}

func TestEmptyChainPipe(t *testing.T) {
	t.Parallel()

	chain := catp.New()

	reader := strings.NewReader("hello\n")
	writer := &strings.Builder{}

	err := chain.Run(reader, writer)
	assert.Equal(t, "", writer.String())
	assert.Nil(t, err)
}

func TestChainProcessPipe(t *testing.T) {
	t.Parallel()

	processConfiguation := catp.ProcessConfiguration{
		[]string{"sed", "s/test/TEST/g"},
		nil,
	}

	chain := catp.New()

	chain.Add(catp.NewProcessPipe(processConfiguation))

	reader := strings.NewReader("test\n")
	writer := &strings.Builder{}

	err := chain.Run(reader, writer)
	assert.Equal(t, "TEST\n", writer.String())
	assert.Nil(t, err)
}

func TestErrorChainProcessPipe(t *testing.T) {
	t.Parallel()

	processConfiguation := catp.ProcessConfiguration{
		[]string{"sh", "-c", "'exit 4'"},
		nil,
	}

	chain := catp.New()

	chain.Add(catp.NewProcessPipe(processConfiguation))

	reader := strings.NewReader("test\n")
	writer := &strings.Builder{}

	err := chain.Run(reader, writer)
	assert.Equal(t, "", writer.String())
	assert.EqualError(t, err, "exit status 127")
}

func TestChainProcessPipeCapturingStderr(t *testing.T) {
	t.Parallel()

	stderrWriter := &strings.Builder{}

	processConfiguation := catp.ProcessConfiguration{
		[]string{"sh", "-c", "echo log 1>&2; cat"},
		stderrWriter,
	}

	chain := catp.New()

	chain.Add(catp.NewProcessPipe(processConfiguation))

	reader := strings.NewReader("test\n")
	writer := &strings.Builder{}

	err := chain.Run(reader, writer)
	assert.Equal(t, "test\n", writer.String())
	assert.Nil(t, err)
	assert.Equal(t, "log\n", stderrWriter.String())
}
