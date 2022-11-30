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
// |          LastConsumerClose |  NewPro. |           | AllConsS.| LastProducerClose            |
// |                            v          v           v          v                              |
// | ,-----------------------------.   ,-------------------.   ,-----------------------------.   |
// | |ConsumerAwaitingAndCloseFirst|   |ConsumerAndProcuder|   |ProducerAwaitingAndCloseFirst|   |
// | |-----------------------------|   |-------------------|   |-----------------------------|   |
// | `-----------------------------'   `-------------------'   `-----------------------------'   |
// |     NewProducer |                    |           |                        | AllConsumersSeen|
// |                 v  LastConsumerClose |           | LastProducerClose      v                 |
// |       ,------------------.           |           |              ,------------------.        |
// |       |ConsumerCloseFirst|           |           |              |ProducerCloseFirst|        |
// +-------|------------------|<----------+           +------------->|------------------|--------+
//         `------------------'                                      `------------------'

import (
	"bufio"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Event int

const (
	ProducerListenerReady Event = iota
	NewProducer
	AllProducersSeen
	ProducerClose
	LastProducerClose

	ConsumerListenerReady
	NewConsumer
	AllConsumersSeen
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

func updateProducers(
	event Event, producers int, poolSize int, producersSeen int,
) (
	newEvent Event, newProducers int, newproducersSeen int,
) {
	newEvent = event
	newProducers = producers
	newproducersSeen = producersSeen
	// nolint: exhaustive
	switch event {
	case NewProducer:
		newProducers = producers + 1
		newproducersSeen = producersSeen + 1

		if poolSize == 0 || newproducersSeen == poolSize {
			newEvent = AllProducersSeen
		}
	case ProducerClose:
		newProducers = producers - 1
		if newProducers == 0 && (poolSize == 0 || producersSeen == poolSize) {
			newEvent = LastProducerClose
		}

	default:
	}

	return newEvent, newProducers, newproducersSeen
}

func updateConsumers(
	event Event, consumers int, poolSize int, consumersSeen int,
) (
	newEvent Event, newConsumers int, newConsumersSeen int,
) {
	newEvent = event
	newConsumers = consumers
	newConsumersSeen = consumersSeen

	// nolint: exhaustive
	switch event {
	case NewConsumer:
		newConsumers = consumers + 1
		newConsumersSeen = consumersSeen + 1

		if poolSize == 0 || newConsumersSeen == poolSize {
			newEvent = AllConsumersSeen
		}

	case ConsumerClose:
		newConsumers = consumers - 1
		if newConsumers == 0 && (poolSize == 0 || consumersSeen == poolSize) {
			newEvent = LastConsumerClose
		}

	default:
	}

	return newEvent, newConsumers, newConsumersSeen
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

	case StateEvent{state: NoConsumerAndNoProducer, event: AllConsumersSeen}:
		newState = ConsumerAwaiting
	case StateEvent{state: ConsumerAwaiting, event: AllProducersSeen}:
		newState = ConsumerAndProcuder

	case StateEvent{state: NoConsumerAndNoProducer, event: AllProducersSeen}:
		newState = ProducerAwaiting
	case StateEvent{state: ProducerAwaiting, event: AllConsumersSeen}:
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
	case StateEvent{state: ProducerAwaitingAndCloseFirst, event: AllConsumersSeen}:
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
	producerNetwork   string
	producerAddress   string
	producersPoolSize int
	consumerNetwork   string
	consumerAddress   string
	consumersPoolSize int
	ch                *ConsumerHandler
	ph                *ProducerHandler
	stream            chan []byte
	quit              chan struct{}
	consumersSeen     int
	producersSeen     int
	interval          time.Duration
}

// New Balancer service.
func New(producerNetwork string, producerAddress string,
	consumerNetwork string, consumerAddress string,
	producersPoolSize int, consumersPoolSize int,
	interval time.Duration) *Balancer {
	return &Balancer{
		RWMutex:           sync.RWMutex{},
		producerNetwork:   producerNetwork,
		producerAddress:   producerAddress,
		producersPoolSize: producersPoolSize,
		consumerNetwork:   consumerNetwork,
		consumerAddress:   consumerAddress,
		consumersPoolSize: consumersPoolSize,
		ch:                &ConsumerHandler{}, // nolint: exhaustivestruct
		ph:                &ProducerHandler{}, // nolint: exhaustivestruct
		stream:            make(chan []byte),
		quit:              make(chan struct{}),
		consumersSeen:     0,
		producersSeen:     0,
		interval:          interval,
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
	b.consumersSeen = 0
	b.producersSeen = 0

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
	log.Info().Int("ConsumersPool", b.consumersPoolSize).Msg("balancer Starting")
	log.Debug().Msg("Lock")
	b.Lock()
	events := make(chan Event, 1)

	b.ph = NewProducerHandler(b, events, b.quit, b.interval)

	b.ch = NewConsumerHandler(b, events, b.quit, b.interval)

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
		event, consumers, b.consumersSeen = updateConsumers(event, consumers, b.consumersPoolSize, b.consumersSeen)
		event, producers, b.producersSeen = updateProducers(event, producers, b.producersPoolSize, b.producersSeen)

		log.Debug().
			Int("consumers", consumers).
			Int("consumersSeen", b.consumersSeen).
			Int("producers", producers).
			Int("producersSeen", b.producersSeen).
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
	conn         net.Conn
	stream       chan []byte
	events       chan<- Event
	quit         <-chan struct{}
	interval     time.Duration
	lineCounter  int
	backPressure bool
}

func (w *ProducerWorker) Start() {
	defer func() {
		w.conn.Close()
		log.Info().Str("remote", w.conn.RemoteAddr().String()).Msg("Producer close connection")
		w.events <- ProducerClose
	}()

	ticker := time.NewTicker(w.interval)
	reader := bufio.NewReader(w.conn)
	incoming := make(chan []byte)
	w.backPressure = false

	go func() {
		for {
			line, err := reader.ReadBytes(byte('\n'))
			if err != nil {
				log.Error().AnErr("readError", err)
				close(incoming)

				break
			}

			incoming <- line
		}
	}()

	for {
		select {
		case in, ok := <-incoming:
			if !ok {
				log.Debug().Str("remote", w.conn.RemoteAddr().String()).Msg("End of stream")

				return
			}

			w.pushInStream(in, ticker.C)
		case <-ticker.C:
			w.updateStats()
			w.backPressure = false
		}
	}
}

func (w *ProducerWorker) updateStats() {
	log.Info().
		Float32("rate l/s",
			(float32(w.lineCounter)*float32(time.Second)/float32(w.interval)),
		).
		Str("remote", w.conn.RemoteAddr().String()).
		Msg("input stream rate")

	w.lineCounter = 0
}

func (w *ProducerWorker) pushInStream(in []byte, updateTicker <-chan time.Time) {
	for {
		select {
		case w.stream <- in:
			w.lineCounter++

			return
		case <-updateTicker:
			if w.backPressure {
				log.Warn().Str("remote", w.conn.RemoteAddr().String()).Msg("Back pressure detected")
			}

			w.updateStats()
			w.backPressure = true
		}
	}
}

type ProducerHandler struct {
	balancer *Balancer
	events   chan<- Event
	quit     <-chan struct{}
	interval time.Duration
}

func NewProducerHandler(
	balancer *Balancer,
	events chan<- Event,
	quit <-chan struct{},
	interval time.Duration) *ProducerHandler {
	return &ProducerHandler{
		balancer: balancer,
		events:   events,
		quit:     quit,
		interval: interval,
	}
}

func (ph *ProducerHandler) handleRequest(conn net.Conn) {
	log.Debug().Msg("RLock")
	ph.balancer.RLock()

	log.Info().Str("remote", conn.RemoteAddr().String()).Msg("New producer")
	ph.events <- NewProducer

	worker := ProducerWorker{
		conn:         conn,
		stream:       ph.balancer.getStream(),
		events:       ph.events,
		quit:         ph.quit,
		interval:     ph.interval,
		lineCounter:  0,
		backPressure: false,
	}
	ph.balancer.RUnlock()
	log.Debug().Msg("RUnLock")
	worker.Start()
}

type ConsumerWorker struct {
	conn     net.Conn
	stream   chan []byte
	events   chan<- Event
	quit     <-chan struct{}
	interval time.Duration
}

func (w *ConsumerWorker) Start() {
	defer func() {
		w.conn.Close()
		log.Info().Str("remote", w.conn.RemoteAddr().String()).Msg("Consumer close connection")
		w.events <- ConsumerClose
	}()

	lineCounter := 0
	ticker := time.NewTicker(w.interval)

	for {
		select {
		case <-ticker.C:
			log.Info().
				Float32("rate l/s", (float32(lineCounter)*float32(time.Second)/float32(w.interval))).
				Str("remote", w.conn.RemoteAddr().String()).
				Msg("output stream rate")

			lineCounter = 0
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
			lineCounter++

		case <-w.quit:
			return
		}
	}
}

type ConsumerHandler struct {
	balancer *Balancer
	events   chan<- Event
	quit     <-chan struct{}
	interval time.Duration
}

func NewConsumerHandler(
	balancer *Balancer,
	events chan<- Event,
	quit <-chan struct{},
	interval time.Duration) *ConsumerHandler {
	return &ConsumerHandler{
		balancer: balancer,
		events:   events,
		quit:     quit,
		interval: interval,
	}
}

func (ch *ConsumerHandler) handleRequest(conn net.Conn) {
	log.Debug().Msg("RLock")
	ch.balancer.RLock()

	log.Info().Str("remote", conn.RemoteAddr().String()).Msg("New consumer")
	ch.events <- NewConsumer

	worker := ConsumerWorker{
		conn:     conn,
		stream:   ch.balancer.getStream(),
		events:   ch.events,
		quit:     ch.quit,
		interval: ch.interval,
	}
	ch.balancer.RUnlock()
	log.Debug().Msg("RUnLock")
	worker.Start()
}
