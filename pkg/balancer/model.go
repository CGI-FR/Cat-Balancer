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
}

// New Balancer service.
func New(producerNetwork string, producerAddress string,
	consumerNetwork string, consumerAddress string) *Balancer {
	return &Balancer{producerNetwork, producerAddress, consumerNetwork, consumerAddress}
}

type Event int

const (
	ProducerListenerReady Event = iota
	NewProducer
	ProducerClose

	ConsumerListenerReady
	NewConsumer
	ConsumerClose
)

type State int

const (
	Initial State = iota
	ProducerReady
	ConsumerReady
	ProducerAndConsumerReady
)

type StateEvent struct {
	state State
	event Event
}

func update(state State, event Event) State {
	switch (StateEvent{state: state, event: event}) {
	case StateEvent{state: Initial, event: ConsumerListenerReady}:
		return ConsumerReady

	case StateEvent{state: Initial, event: ProducerListenerReady}:
		return ProducerReady

	case StateEvent{state: ConsumerReady, event: ProducerListenerReady}:
		return ProducerAndConsumerReady
	case StateEvent{state: ProducerReady, event: ConsumerListenerReady}:
		return ProducerAndConsumerReady

	default:
		log.Logger.Info().Int("state", int(state)).Int("event", int(event)).Msg("Unknow state and event action")

		return state
	}
}

// Start the balancer Service and wait for clients.
func (b Balancer) Start() {
	stream := make(chan []byte, 1)
	events := make(chan Event, 1)

	producer := NewListenerFactory(
		NewProducerHandler(stream, events),
		b.producerNetwork,
		b.producerAddress,
		func() { events <- ProducerListenerReady },
	)

	log.Info().Str("producerAddress", b.producerAddress).Msg("Starting Producer Server")

	go producer.Start()

	consumer := NewListenerFactory(
		NewConsumerHandler(stream, events),
		b.consumerNetwork, b.consumerAddress,
		func() { events <- ConsumerListenerReady },
	)

	log.Info().Str("ConsumerAddress", b.consumerAddress).Msg("Starting Consumer Server")

	go consumer.Start()

	state := Initial

	for event := range events {
		state = update(state, event)
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
	events chan<- Event
}

func NewProducerHandler(stream chan<- []byte, events chan<- Event) ProducerHandler {
	return ProducerHandler{stream, events}
}

func (ph ProducerHandler) handleRequest(conn net.Conn) {
	log.Info().IPAddr("remote", net.IP(conn.RemoteAddr().Network())).Msg("New Producer")
	ph.events <- NewProducer

	defer func() {
		conn.Close()
		log.Info().IPAddr("remote", net.IP(conn.RemoteAddr().Network())).Msg("Producer close connection")
		ph.events <- ProducerClose
	}()

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
	events chan<- Event
}

func NewConsumerHandler(stream <-chan []byte, events chan<- Event) ConsumerHandler {
	return ConsumerHandler{stream, events}
}

func (ch ConsumerHandler) handleRequest(conn net.Conn) {
	log.Info().IPAddr("remote", net.IP(conn.RemoteAddr().Network())).Msg("New consumer")
	ch.events <- NewConsumer

	defer func() {
		conn.Close()
		log.Info().IPAddr("remote", net.IP(conn.RemoteAddr().Network())).Msg("Consumer close connection")
		ch.events <- ConsumerClose
	}()

	for line := range ch.stream {
		_, err := conn.Write(line)
		if err != nil {
			log.Err(err)

			break
		}
	}
}
