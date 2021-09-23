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

package balancer_test

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/cgi-fr/cat-balancer/pkg/balancer"
	"github.com/stretchr/testify/assert"
)

func getFreePorts(n int) ([]int, error) {
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

func TestBalancerStart(t *testing.T) {
	t.Parallel()

	ports, err := getFreePorts(2)
	if err != nil {
		t.Error(err)
	}

	b := balancer.New("tcp", fmt.Sprintf(":%d", ports[0]), "tcp", fmt.Sprintf(":%d", ports[1]))

	go b.Start()

	time.Sleep(time.Second)

	producer, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", ports[0]))
	if err != nil {
		t.Fatal("could not connect to producer server: ", err)
	}
	defer producer.Close()

	consumer, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", ports[1]))
	if err != nil {
		t.Fatal("could not connect to consumer server: ", err)
	}
	defer consumer.Close()

	_, err = producer.Write([]byte("hello world\n"))

	if err != nil {
		t.Fatal("could not write in producer stream", err)
	}

	reader := bufio.NewReader(consumer)

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal("could not read in consumer stream", err)
	}

	assert.Equal(t, "hello world\n", line)
}

func consume(t *testing.T, port int, inputs chan string, wg *sync.WaitGroup) {
	t.Helper()

	consumer, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Error("could not connect to consumer server: ", err)
	}

	defer func() {
		consumer.Close()
		wg.Done()
	}()

	for {
		reader := bufio.NewReader(consumer)

		line, err := reader.ReadString('\n')
		if err != nil {
			assert.Equal(t, "EOF", err.Error())

			return
		}

		inputs <- line
	}
}

func TestManyConsumersOneProducer(t *testing.T) {
	t.Parallel()

	const (
		CONSUMER int = 10
		MESSAGES int = 100
	)

	ports, _ := getFreePorts(2)

	b := balancer.New("tcp", fmt.Sprintf(":%d", ports[0]), "tcp", fmt.Sprintf(":%d", ports[1]))

	go b.Start()

	time.Sleep(time.Second)

	producer, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", ports[0]))
	if err != nil {
		t.Fatal("could not connect to producer server: ", err)
	}
	defer producer.Close()

	inputs := make(chan string, 1)

	var wg sync.WaitGroup

	wg.Add(CONSUMER)

	for i := 0; i < CONSUMER; i++ {
		go consume(t, ports[1], inputs, &wg)
	}

	go func() {
		for i := 0; i < CONSUMER*MESSAGES; i++ {
			line := <-inputs

			assert.Equal(t, "hello world\n", line)
		}
	}()

	for i := 0; i < CONSUMER*MESSAGES; i++ {
		_, err = producer.Write([]byte("hello world\n"))
		if err != nil {
			t.Fatal("could not write in producer stream", err)
		}
	}

	time.Sleep(time.Second)
	producer.Close()

	wg.Wait()
}
