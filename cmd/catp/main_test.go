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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cgi-fr/cat-balancer/pkg/balancer"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const (
	producerPort = 1691
	consumerPort = 1961
)

func Test_run(t *testing.T) {
	t.Parallel()

	go balancer.New("tcp", fmt.Sprintf(":%d", producerPort), "tcp", fmt.Sprintf(":%d", consumerPort)).Start()

	time.Sleep(1 * time.Second)

	wg := sync.WaitGroup{}

	wg.Add(1)

	go func() {
		reader := strings.NewReader("hello\n")
		// nolint: exhaustivestruct
		cmd := &cobra.Command{}
		cmd.SetIn(reader)

		err := run(cmd, "", "127.0.0.1:1691")

		assert.Nil(t, err)
		wg.Done()
	}()

	wg.Add(1)

	go func() {
		writer := strings.Builder{}
		// nolint: exhaustivestruct
		cmd := &cobra.Command{}
		cmd.SetOut(&writer)

		err := run(cmd, "127.0.0.1:1961", "")
		assert.Nil(t, err)
		assert.Equal(t, "hello\n", writer.String())
		wg.Done()
	}()

	wg.Wait()
}
