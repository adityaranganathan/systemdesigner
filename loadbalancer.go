package main

import (
	"context"
	"github.com/lithammer/shortuuid/v3"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type LoadBalancer struct {
	ID       string
	Type     string
	Position Position

	InRequests  chan Request
	InResponses chan Response
	Targets     []Target

	messages     chan Message
	numProcessed int64
}

type Target struct {
	OutRequests  chan Request
	OutResponses chan Response
}

func NewLoadBalancer(messages chan Message) *LoadBalancer {
	return &LoadBalancer{
		ID:       shortuuid.New(),
		Type:     LoadBalancerType,
		messages: messages,
	}
}

func (lb *LoadBalancer) GetID() string {
	return lb.ID
}

func (lb *LoadBalancer) GetType() string {
	return lb.Type
}

func (lb *LoadBalancer) GetPosition() Position {
	return lb.Position
}

func (lb *LoadBalancer) SetPosition(pos Position) {
	lb.Position = pos
}

func (lb *LoadBalancer) SetInChannels(inRequests chan Request, inResponses chan Response) {
	lb.InRequests = inRequests
	lb.InResponses = inResponses
}

func (lb *LoadBalancer) SetOutChannels(outRequests chan Request, outResponses chan Response) {
	lb.Targets = append(lb.Targets, Target{
		OutRequests:  outRequests,
		OutResponses: outResponses,
	})
}

func (lb *LoadBalancer) Run(ctx context.Context, wg *sync.WaitGroup, _ context.CancelFunc) {
	metricsTicker := time.NewTicker(100 * time.Millisecond)

	wg.Add(1)
	go func() {
		defer wg.Done()

		targetNumber := 0
		for {
			select {
			case <-ctx.Done():
				log.Printf("%s recieving goroutine cancelled", lb.Type)
				return
			case request := <-lb.InRequests:
				log.Printf("%s forwarding request from client to target #%d", lb.Type, targetNumber)
				lb.Targets[targetNumber].OutRequests <- request
				targetNumber++
				if targetNumber == len(lb.Targets) {
					targetNumber = 0
				}
			case _ = <-metricsTicker.C:
				log.Printf("%s sending metrics: Processed = %d, Queued = %d", lb.Type, int(lb.numProcessed), len(lb.InRequests))

				lb.messages <- Message{
					NodeID: lb.ID,
					Metrics: []Metric{
						NewProcessed(int(lb.numProcessed)),
						NewQueued(len(lb.InRequests)),
					},
				}
			}
		}
	}()

	for i, target := range lb.Targets {
		wg.Add(1)
		go func(targetNumber int, target Target) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					log.Printf("%s target #%d goroutine cancelled", lb.Type, targetNumber)
					return
				case response := <-target.OutResponses:
					log.Printf("%s forwarding response from target #%d to client", lb.Type, targetNumber)
					lb.InResponses <- response
					atomic.AddInt64(&lb.numProcessed, 1)
				}
			}
		}(i, target)
	}
}

func (lb *LoadBalancer) Reset() {
	lb.Targets = []Target{}
	lb.numProcessed = 0
}

func (lb *LoadBalancer) GetMetrics() []Metric {
	return []Metric{
		NewProcessed(0),
		NewQueued(0),
	}
}
