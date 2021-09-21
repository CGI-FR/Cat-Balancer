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
	"net"
	"testing"
	"time"

	"github.com/cgi-fr/cat-balancer/pkg/balancer"
	"github.com/stretchr/testify/assert"
)

// nolint: paralleltest
func TestBalancerStart(t *testing.T) {
	b := balancer.New("tcp", ":1123", "tcp", ":1124")

	go b.Start()

	time.Sleep(time.Second)

	producer, err := net.Dial("tcp", "localhost:1123")
	if err != nil {
		t.Fatal("could not connect to producer server: ", err)
	}
	defer producer.Close()

	consumer, err := net.Dial("tcp", "localhost:1124")
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

// nolint: paralleltest
func TestManyConsumersOneProducer(t *testing.T) {
	b := balancer.New("tcp", ":1125", "tcp", ":1126")

	go b.Start()

	time.Sleep(time.Second)

	producer, err := net.Dial("tcp", "localhost:1125")
	if err != nil {
		t.Fatal("could not connect to producer server: ", err)
	}
	defer producer.Close()

	const CONSUMER int = 10

	inputs := make(chan string, 1)

	for i := 0; i < CONSUMER; i++ {
		go func() {
			consumer, err := net.Dial("tcp", "localhost:1126")
			if err != nil {
				t.Error("could not connect to consumer server: ", err)
			}
			defer consumer.Close()

			reader := bufio.NewReader(consumer)

			line, err := reader.ReadString('\n')
			if err != nil {
				t.Error("could not read in consumer stream", err)

				return
			}

			inputs <- line
		}()
	}

	_, err = producer.Write([]byte("hello world\n"))

	if err != nil {
		t.Fatal("could not write in producer stream", err)
	}

	line := <-inputs
	assert.Equal(t, "hello world\n", line)
}
