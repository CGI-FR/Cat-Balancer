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

// nolint: testpackage
package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/cgi-fr/cat-balancer/pkg/balancer"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// GetFreePorts return free port.
func GetFreePorts(n int) ([]int, error) {
	ports := make([]int, n)

	for k := range ports {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			// nolint: wrapcheck
			return ports, err
		}

		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			// nolint: wrapcheck
			return ports, err
		}
		// This is done on purpose - we want to keep ports
		// busy to avoid collisions when getting the next one
		defer func() { _ = l.Close() }()

		ports[k] = l.Addr().(*net.TCPAddr).Port
	}

	return ports, nil
}

func Test_run(t *testing.T) {
	ports, err := GetFreePorts(2)
	if err != nil {
		t.Error(err)
	}

	var (
		producerPort = ports[0]
		consumerPort = ports[1]
	)

	t.Parallel()

	go balancer.New("tcp", fmt.Sprintf(":%d", producerPort), "tcp", fmt.Sprintf(":%d", consumerPort)).Start()

	wg := sync.WaitGroup{}

	wg.Add(1)

	go func() {
		reader := strings.NewReader("hello\n")
		// nolint: exhaustivestruct
		cmd := &cobra.Command{}
		cmd.SetIn(reader)

		err := run(cmd, "", fmt.Sprintf("127.0.0.1:%d", producerPort), []string{}, "", "")

		assert.Nil(t, err)
		wg.Done()
	}()

	wg.Add(1)

	go func() {
		writer := strings.Builder{}
		// nolint: exhaustivestruct
		cmd := &cobra.Command{}
		cmd.SetOut(&writer)

		err := run(cmd, fmt.Sprintf("127.0.0.1:%d", consumerPort), "", []string{}, "", "")
		assert.Nil(t, err)
		assert.Equal(t, "hello\n", writer.String())
		wg.Done()
	}()

	wg.Wait()
}

func Test_run_producer_first(t *testing.T) {
	ports, err := GetFreePorts(2)
	if err != nil {
		t.Error(err)
	}

	var (
		producerPort = ports[0]
		consumerPort = ports[1]
	)

	t.Parallel()

	go balancer.New("tcp", fmt.Sprintf(":%d", producerPort), "tcp", fmt.Sprintf(":%d", consumerPort)).Start()

	reader := strings.NewReader("hello\n")
	// nolint: exhaustivestruct
	cmd := &cobra.Command{}
	cmd.SetIn(reader)

	err = run(cmd, "", fmt.Sprintf("127.0.0.1:%d", producerPort), []string{}, "", "")

	assert.Nil(t, err)

	writer := strings.Builder{}
	// nolint: exhaustivestruct
	cmd = &cobra.Command{}
	cmd.SetOut(&writer)

	err = run(cmd, fmt.Sprintf("127.0.0.1:%d", consumerPort), "", []string{}, "", "")
	assert.Nil(t, err)
	assert.Equal(t, "hello\n", writer.String())
}
