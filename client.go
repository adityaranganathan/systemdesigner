package main

import (
	"context"
	"github.com/lithammer/shortuuid/v3"
	"log"
	"sync"
	"time"
)

const (
	NumRequests     = 1000
	RequestInterval = 10 * time.Millisecond
)

type Client struct {
	ID       string
	Type     string
	Position Position

	outRequests  chan Request
	outResponses chan Response

	requestStore *RequestStore
	messages     chan Message
}

func NewClient(messages chan Message) *Client {
	return &Client{
		ID:           shortuuid.New(),
		messages:     messages,
		Type:         ClientType,
		requestStore: NewRequestStore(),
	}
}

func (c *Client) GetID() string {
	return c.ID
}

func (c *Client) GetType() string {
	return c.Type
}

func (c *Client) GetPosition() Position {
	return c.Position
}

func (c *Client) SetPosition(pos Position) {
	c.Position = pos
}

func (c *Client) SetOutChannels(outRequests chan Request, outResponses chan Response) {
	c.outRequests = outRequests
	c.outResponses = outResponses
}

func (c *Client) Run(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc) {
	var startTime time.Time

	wg.Add(1)
	go func() {
		defer wg.Done()

		startTime = time.Now()
		for i := 0; i < NumRequests; i++ {
			log.Printf("%s sending request number %d", c.Type, i)
			select {
			case <-ctx.Done():
				log.Printf("%s request goroutine cancelled", c.Type)
				return
			default:
				newRequest := Request{ID: i, SentAt: time.Now()}
				c.outRequests <- newRequest
				c.requestStore.Put(i, newRequest)
				time.Sleep(RequestInterval)
			}
		}
		log.Printf("%s request goroutine complete", c.Type)
	}()

	metricsTicker := time.NewTicker(100 * time.Millisecond)

	wg.Add(1)
	go func() {
		defer func() {
			metricsTicker.Stop()
			wg.Done()
		}()

		totalLatency := 0
		numResponses := 0
		for {
			select {
			case <-ctx.Done():
				log.Printf("%s response goroutine cancelled", c.Type)
				return
			case response := <-c.outResponses:
				numResponses += 1
				response.ReceivedAt = time.Now()
				latency := response.ReceivedAt.Sub(c.requestStore.Get(response.ID).SentAt)
				totalLatency += int(latency.Milliseconds())

				log.Printf("%s received response for request %d. Latency = %d, Avg. Latency = %d", c.Type, response.ID, latency.Milliseconds(), totalLatency/numResponses)

			case _ = <-metricsTicker.C:
				duration := int(time.Since(startTime).Seconds())
				if duration == 0 || numResponses == 0 {
					log.Printf("%s not sending metrics", c.Type)
					continue
				}

				log.Printf("%s sending metrics: Num Responses = %d, Avg. Latency = %d", c.Type, numResponses, totalLatency/numResponses)
				c.messages <- Message{
					NodeID: c.ID,
					Metrics: []Metric{
						NewNumResponses(numResponses),
						NewAvgLatency(totalLatency / numResponses),
					},
				}

				if numResponses == NumRequests {
					log.Printf("%s response goroutine complete", c.Type)
					cancel()
					return
				}
			}
		}
	}()
}

func (c *Client) Reset() {
	c.requestStore = NewRequestStore()
}

func (c *Client) GetMetrics() []Metric {
	return []Metric{
		NewNumResponses(0),
		NewAvgLatency(0),
	}
}

type RequestStore struct {
	m     map[int]Request
	mutex sync.RWMutex
}

func NewRequestStore() *RequestStore {
	return &RequestStore{
		m:     map[int]Request{},
		mutex: sync.RWMutex{},
	}
}

func (m *RequestStore) Put(id int, req Request) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.m[id] = req
}

func (m *RequestStore) Get(id int) Request {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.m[id]
}
