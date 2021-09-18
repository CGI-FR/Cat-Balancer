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

package balancer

import (
	"bufio"
	"net"

	"github.com/rs/zerolog/log"
)

type ListenerFactory func() Listener

type Connexion interface{}

type Balancer struct {
	producerNetwork string
	producerAddress string
	consumerNetwork string
	consumerAddress string
	controler       chan Control
}

// New Balancer service.
func New(producerNetwork string, producerAddress string,
	consumerNetwork string, consumerAddress string) *Balancer {
	return &Balancer{producerNetwork, producerAddress, consumerNetwork, consumerAddress, nil}
}

type Control int

const (
	ProducerListenerReady Control = iota
	ConsumerListenerReady
)

// Start the balancer Service and wait for clients.
func (b Balancer) Start() {
	stream := make(chan []byte, 1)
	b.controler = make(chan Control, 1)

	producer := NewListenerFactory(
		NewProducerHandler(stream),
		b.producerNetwork,
		b.producerAddress,
		func() { b.controler <- ProducerListenerReady },
	)

	log.Info().Str("producerAddress", b.producerAddress).Msg("Starting Producer Server")

	go producer.Start()

	consumer := NewListenerFactory(
		NewConsumerHandler(stream),
		b.consumerNetwork, b.consumerAddress,
		func() { b.controler <- ConsumerListenerReady },
	)

	log.Info().Str("ConsumerAddress", b.consumerAddress).Msg("Starting Consumer Server")

	go consumer.Start()

	producerReady := false
	consumerReady := false

	for control := range b.controler {
		switch control {
		case ProducerListenerReady:
			producerReady = true
		case ConsumerListenerReady:
			consumerReady = true
		}

		if consumerReady && producerReady {
			log.Info().Msg("Balancer is awaiting clients")

			return
		}
	}
}

type Handler interface {
	handleRequest(net.Conn)
}

type Listener struct {
	Handler
	network       string
	address       string
	readyCallback func()
}

func NewListenerFactory(handler Handler, network string, address string, readyCallback func()) *Listener {
	return &Listener{handler, network, address, readyCallback}
}

func (l Listener) Start() {
	// Listen for incoming connections.
	listen, err := net.Listen(l.network, l.address)
	if err != nil {
		log.Error().AnErr("Error listening:", err)

		return
	}

	l.readyCallback()
	// Close the listener when the application closes.
	defer listen.Close()

	for {
		// Listen for an incoming connection.
		conn, err := listen.Accept()
		if err != nil {
			log.Error().AnErr("Error accecpting", err)
		}
		// Handle connections in a new goroutine.
		go l.Handler.handleRequest(conn)
	}
}

type ProducerHandler struct {
	stream chan<- []byte
}

func NewProducerHandler(stream chan<- []byte) ProducerHandler {
	return ProducerHandler{stream}
}

func (ph ProducerHandler) handleRequest(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			log.Error().AnErr("readError", err)

			break
		}
		ph.stream <- line
	}
}

type ConsumerHandler struct {
	stream <-chan []byte
}

func NewConsumerHandler(stream <-chan []byte) ConsumerHandler {
	return ConsumerHandler{stream}
}

func (ch ConsumerHandler) handleRequest(conn net.Conn) {
	defer func() {
		conn.Close()
		log.Info().IPAddr("remote", net.IP(conn.RemoteAddr().Network())).Msg("Consumer close connection")
	}()

	for line := range ch.stream {
		_, err := conn.Write(line)
		if err != nil {
			log.Err(err)

			break
		}
	}
}
