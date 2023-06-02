package main

import (
	"context"
	"github.com/lithammer/shortuuid/v3"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxRoutines         = 20
	ProcessingTimeLower = 300
	ProcessingTimeUpper = 600
)

type Server struct {
	ID       string
	Type     string
	Position Position

	InRequests  chan Request
	InResponses chan Response
	routines    chan int

	messages     chan Message
	numProcessed int64
}

func NewServer(messages chan Message) *Server {
	routines := make(chan int, MaxRoutines)
	return &Server{
		ID:       shortuuid.New(),
		Type:     ServerType,
		routines: routines,
		messages: messages,
	}
}

func (s *Server) GetID() string {
	return s.ID
}

func (s *Server) GetType() string {
	return s.Type
}

func (s *Server) GetPosition() Position {
	return s.Position
}

func (s *Server) SetPosition(pos Position) {
	s.Position = pos
}

func (s *Server) SetInChannels(inRequests chan Request, inResponses chan Response) {
	s.InRequests = inRequests
	s.InResponses = inResponses
}

func (s *Server) Run(ctx context.Context, wg *sync.WaitGroup, _ context.CancelFunc) {
	metricsTicker := time.NewTicker(100 * time.Millisecond)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				log.Printf("%s recieving goroutine cancelled", s.Type)
				return
			case request := <-s.InRequests:
				s.routines <- 1

				wg.Add(1)
				go s.Process(ctx, wg, request)
			case _ = <-metricsTicker.C:
				utilization := int(float64(len(s.routines)) / float64(MaxRoutines) * 100)
				log.Printf("%s sending metrics: Processed = %d, Queued = %d, Utilisation = %v", s.Type, int(s.numProcessed), len(s.InRequests), utilization)

				s.messages <- Message{
					NodeID: s.ID,
					Metrics: []Metric{
						NewProcessed(int(s.numProcessed)),
						NewQueued(len(s.InRequests)),
						NewUtilisation(utilization),
					},
				}
			}
		}
	}()
}

func (s *Server) Process(ctx context.Context, wg *sync.WaitGroup, request Request) {
	defer func() {
		<-s.routines
		wg.Done()
	}()

	select {
	case <-ctx.Done():
		log.Printf("%s processing request number %d cancelled", s.Type, request.ID)
		return
	default:
		log.Printf("%s receieved request number %d", s.Type, request.ID)

		rand.Seed(time.Now().UnixNano())
		// Generate a random duration between 200 and 500 milliseconds
		processingTime := time.Duration(rand.Intn(ProcessingTimeUpper-ProcessingTimeLower+1)+ProcessingTimeLower) * time.Millisecond
		time.Sleep(processingTime)

		log.Printf("%s responding to request number %d", s.Type, request.ID)
		s.InResponses <- Response{ID: request.ID}
		atomic.AddInt64(&s.numProcessed, 1)
	}
}

func (s *Server) Reset() {
	s.routines = make(chan int, MaxRoutines)
	s.numProcessed = 0
}

func (s *Server) GetMetrics() []Metric {
	return []Metric{
		NewProcessed(0),
		NewQueued(0),
		NewUtilisation(0),
	}
}
