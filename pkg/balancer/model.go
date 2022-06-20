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

//
//                                           ,-------.
//                                           |Initial|
//                                      +----|-------|---+
//                ProducerListenerReady |    `-------'   | ConsumerListenerReady
//                                      v                v
//                               ,-------------.   ,-------------.
//                               |ProducerReady|   |ConsumerReady|
//                               |-------------|   |-------------|
//                               `-------------'   `-------------'
//                ConsumerListenerReady |                | ProducerListenerReady
//                                      v                v
//                                   ,-----------------------.
//                 LastProducerClose |NoConsumerAndNoProducer| LastConsumerClose
// +-------------------------------->|-----------------------|<----------------------------------+
// |                                 `-----------------------'                                   |
// |                       NewConsumer |                 | NewProducer                           |
// |                                   v                 v                                       |
// |                          ,----------------.   ,----------------.                            |
// |                          |ConsumerAwaiting|   |ProducerAwaiting|                            |
// |                          |----------------|   |----------------|                            |
// |                          `----------------'   `----------------'                            |
// |          LastProducerClose |  NewPro. |           | NewCons. | LastConsumerClose            |
// |                            v          v           v          v                              |
// | ,-----------------------------.   ,-------------------.   ,-----------------------------.   |
// | |ConsumerAwaitingAndCloseFirst|   |ConsumerAndProcuder|   |ProducerAwaitingAndCloseFirst|   |
// | |-----------------------------|   |-------------------|   |-----------------------------|   |
// | `-----------------------------'   `-------------------'   `-----------------------------'   |
// |     NewProducer |                    |           |                        | NewConsumer     |
// |                 v  LastConsumerClose |           | LastProducerClose      v                 |
// |       ,------------------.           |           |              ,------------------.        |
// |       |ConsumerCloseFirst|           |           |              |ProducerCloseFirst|        |
// +-------|------------------|<----------+           +------------->|------------------|--------+
//         `------------------'                                      `------------------'

import (
	"bufio"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
)

type Event int

const (
	ProducerListenerReady Event = iota
	NewProducer
	ProducerClose
	LastProducerClose

	ConsumerListenerReady
	NewConsumer
	ConsumerClose
	LastConsumerClose
)

type State int

const (
	Initial State = iota
	ProducerReady
	ConsumerReady
	NoConsumerAndNoProducer
	ConsumerAwaiting
	ProducerAwaiting
	ConsumerAndProcuder
	ConsumerCloseFirst
	ProducerCloseFirst
	ConsumerAwaitingAndCloseFirst
	ProducerAwaitingAndCloseFirst
)

type StateEvent struct {
	state State
	event Event
}

func updateClients(event Event, consumers int, producers int) (newEvent Event, newConsumers int, newProducers int) {
	newEvent = event
	newConsumers = consumers
	newProducers = producers

	// nolint: exhaustive
	switch event {
	case NewConsumer:
		newConsumers = consumers + 1
	case NewProducer:
		newProducers = producers + 1
	case ConsumerClose:
		newConsumers = consumers - 1
		if newConsumers == 0 {
			newEvent = LastConsumerClose
		}
	case ProducerClose:
		newProducers = producers - 1
		if newProducers == 0 {
			newEvent = LastProducerClose
		}

	default:
	}

	return newEvent, newConsumers, newProducers
}

// nolint: cyclop
func update(
	state State, event Event,
) (
	newState State,
) {
	switch (StateEvent{state: state, event: event}) {
	case StateEvent{state: Initial, event: ConsumerListenerReady}:
		newState = ConsumerReady

	case StateEvent{state: Initial, event: ProducerListenerReady}:
		newState = ProducerReady

	case StateEvent{state: ConsumerReady, event: ProducerListenerReady}:
		newState = NoConsumerAndNoProducer
	case StateEvent{state: ProducerReady, event: ConsumerListenerReady}:
		newState = NoConsumerAndNoProducer

	case StateEvent{state: NoConsumerAndNoProducer, event: NewConsumer}:
		newState = ConsumerAwaiting
	case StateEvent{state: ConsumerAwaiting, event: NewProducer}:
		newState = ConsumerAndProcuder

	case StateEvent{state: NoConsumerAndNoProducer, event: NewProducer}:
		newState = ProducerAwaiting
	case StateEvent{state: ProducerAwaiting, event: NewConsumer}:
		newState = ConsumerAndProcuder

	case StateEvent{state: ConsumerAndProcuder, event: LastConsumerClose}:
		newState = ConsumerCloseFirst
	case StateEvent{state: ConsumerCloseFirst, event: LastProducerClose}:
		newState = NoConsumerAndNoProducer

	case StateEvent{state: ConsumerAndProcuder, event: LastProducerClose}:
		newState = ProducerCloseFirst
	case StateEvent{state: ProducerCloseFirst, event: LastConsumerClose}:
		newState = NoConsumerAndNoProducer

	case StateEvent{state: ProducerAwaiting, event: LastProducerClose}:
		newState = ProducerAwaitingAndCloseFirst
	case StateEvent{state: ProducerAwaitingAndCloseFirst, event: NewConsumer}:
		newState = ProducerCloseFirst

	case StateEvent{state: ConsumerAwaiting, event: LastProducerClose}:
		newState = ConsumerAwaitingAndCloseFirst
	case StateEvent{state: ConsumerAwaitingAndCloseFirst, event: NewProducer}:
		newState = ConsumerCloseFirst

	default:
		log.Logger.Warn().Int("state", int(state)).Int("event", int(event)).Msg("Unknow state and event action")

		newState = state
	}

	return newState
}

type Balancer struct {
	sync.RWMutex
	producerNetwork string
	producerAddress string
	consumerNetwork string
	consumerAddress string
	ch              *ConsumerHandler
	ph              *ProducerHandler
	stream          chan []byte
	quit            chan struct{}
}

// New Balancer service.
func New(producerNetwork string, producerAddress string,
	consumerNetwork string, consumerAddress string) *Balancer {
	return &Balancer{
		RWMutex:         sync.RWMutex{},
		producerNetwork: producerNetwork,
		producerAddress: producerAddress,
		consumerNetwork: consumerNetwork,
		consumerAddress: consumerAddress,
		ch:              &ConsumerHandler{}, // nolint: exhaustivestruct
		ph:              &ProducerHandler{}, // nolint: exhaustivestruct
		stream:          make(chan []byte),
		quit:            make(chan struct{}),
	}
}

func (b *Balancer) action(state State) {
	// nolint: exhaustive
	switch state {
	case NoConsumerAndNoProducer:
		b.resetSession()
	case ProducerCloseFirst:
		b.closeStream()
	}
}

func (b *Balancer) resetSession() {
	b.stream = make(chan []byte, 1)

	log.Logger.Info().Msg("Start to a new stream")

	b.Unlock()
	log.Debug().Msg("Unlock")
}

func (b *Balancer) closeStream() {
	log.Debug().Msg("Lock")
	log.Info().Msg("Drain current stream")
	b.Lock()
	close(b.stream)
	log.Debug().Msg("Close current Stream")
}

// Start the balancer Service and wait for clients.
func (b *Balancer) Start() {
	log.Debug().Msg("Lock")
	b.Lock()
	events := make(chan Event, 1)

	b.ph = NewProducerHandler(b, events, b.quit)

	b.ch = NewConsumerHandler(b, events, b.quit)

	producer := NewListenerFactory(
		b.ph,
		b.producerNetwork,
		b.producerAddress,
		func() { events <- ProducerListenerReady },
	)

	log.Info().Str("producerAddress", b.producerAddress).Msg("Starting Producer Server")

	go producer.Start()

	consumer := NewListenerFactory(
		b.ch,
		b.consumerNetwork, b.consumerAddress,
		func() { events <- ConsumerListenerReady },
	)

	log.Info().Str("ConsumerAddress", b.consumerAddress).Msg("Starting Consumer Server")

	go consumer.Start()

	state := Initial
	consumers := 0
	producers := 0

	for event := range events {
		log.Debug().Int("state", int(state)).Int("event", int(event)).Msg("Start update")
		event, consumers, producers = updateClients(event, consumers, producers)

		log.Debug().
			Int("consumers", consumers).
			Int("producers", producers).
			Int("computedEvent", int(event)).
			Msg("consumers producer stats")

		newState := update(state, event)

		if newState != state {
			b.action(newState)
			state = newState
		}

		log.Debug().Int("state", int(state)).Msg("End update")
	}
}

func (b *Balancer) getStream() chan []byte {
	log.Debug().Msg("Get Current stream")

	return b.stream
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

type ProducerWorker struct {
	conn   net.Conn
	stream chan []byte
	events chan<- Event
	quit   <-chan struct{}
}

func (w *ProducerWorker) Start() {
	defer func() {
		w.conn.Close()
		log.Info().Str("remote", w.conn.RemoteAddr().String()).Msg("Producer close connection")
		w.events <- ProducerClose
	}()

	reader := bufio.NewReader(w.conn)

	for {
		line, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			log.Error().AnErr("readError", err)

			break
		}

		w.stream <- line
	}
}

type ProducerHandler struct {
	balancer *Balancer
	events   chan<- Event
	quit     <-chan struct{}
}

func NewProducerHandler(balancer *Balancer, events chan<- Event, quit <-chan struct{}) *ProducerHandler {
	return &ProducerHandler{balancer: balancer, events: events, quit: quit}
}

func (ph *ProducerHandler) handleRequest(conn net.Conn) {
	log.Debug().Msg("RLock")
	ph.balancer.RLock()

	log.Info().Str("remote", conn.RemoteAddr().String()).Msg("New producer")
	ph.events <- NewProducer

	worker := ProducerWorker{conn: conn, stream: ph.balancer.getStream(), events: ph.events, quit: ph.quit}
	ph.balancer.RUnlock()
	log.Debug().Msg("RUnLock")
	worker.Start()
}

type ConsumerWorker struct {
	conn   net.Conn
	stream chan []byte
	events chan<- Event
	quit   <-chan struct{}
}

func (w *ConsumerWorker) Start() {
	defer func() {
		w.conn.Close()
		log.Info().Str("remote", w.conn.RemoteAddr().String()).Msg("Consumer close connection")
		w.events <- ConsumerClose
	}()

	for {
		select {
		case line, ok := <-w.stream:
			if !ok {
				log.Debug().Str("remote", w.conn.RemoteAddr().String()).Msg("End of stream")

				return
			}

			_, err := w.conn.Write(line)
			if err != nil {
				log.Err(err)

				return
			}
		case <-w.quit:
			return
		}
	}
}

type ConsumerHandler struct {
	balancer *Balancer
	events   chan<- Event
	quit     <-chan struct{}
}

func NewConsumerHandler(balancer *Balancer, events chan<- Event, quit <-chan struct{}) *ConsumerHandler {
	return &ConsumerHandler{balancer: balancer, events: events, quit: quit}
}

func (ch *ConsumerHandler) handleRequest(conn net.Conn) {
	log.Debug().Msg("RLock")
	ch.balancer.RLock()

	log.Info().Str("remote", conn.RemoteAddr().String()).Msg("New consumer")
	ch.events <- NewConsumer

	worker := ConsumerWorker{conn: conn, stream: ch.balancer.getStream(), events: ch.events, quit: ch.quit}
	ch.balancer.RUnlock()
	log.Debug().Msg("RUnLock")
	worker.Start()
}
