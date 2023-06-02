package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/lithammer/shortuuid/v3"
	"log"
	"sync"
	"time"
)

type Node interface {
	GetID() string
	GetType() string
	GetMetrics() []Metric
	GetPosition() Position
	SetPosition(Position)
	Run(context.Context, *sync.WaitGroup, context.CancelFunc)
	Reset()
}

type Edge struct {
	ID       string
	SourceID string
	TargetID string
}

type Sender interface {
	SetOutChannels(chan Request, chan Response)
}

type Receiver interface {
	SetInChannels(chan Request, chan Response)
}

type Request struct {
	ID     int
	SentAt time.Time
}

type Response struct {
	ID         int
	ReceivedAt time.Time
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

const (
	ClientType       string = "client"
	ServerType       string = "server"
	LoadBalancerType string = "load balancer"
)

type System struct {
	ID        string
	nodeStore map[string]Node
	edgeStore map[string]Edge

	messages chan Message

	isResetting bool

	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         *sync.WaitGroup
}

func NewSystem() *System {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	messages := make(chan Message)

	return &System{
		ID:          shortuuid.New(),
		nodeStore:   map[string]Node{},
		edgeStore:   map[string]Edge{},
		messages:    messages,
		isResetting: false,
		ctx:         ctx,
		cancelFunc:  cancel,
		wg:          wg,
	}
}

func (s *System) AddNode(nodeType string) Node {
	var node Node
	switch nodeType {
	case ClientType:
		node = NewClient(s.messages)
	case ServerType:
		node = NewServer(s.messages)
	case LoadBalancerType:
		node = NewLoadBalancer(s.messages)
	}
	node.SetPosition(Position{
		X: 500,
		Y: 150,
	})
	s.nodeStore[node.GetID()] = node

	return node
}

func (s *System) Start() error {
	if !s.isResetting {
		log.Printf("system %s reset intitiated", s.ID)

		s.cancelFunc()
		s.isResetting = true
		s.wg.Wait()

		wg := &sync.WaitGroup{}
		s.wg = wg

		ctx, cancel := context.WithCancel(context.Background())
		s.ctx = ctx
		s.cancelFunc = cancel

		log.Printf("system %s reset complete", s.ID)
		s.isResetting = false
	} else {
		log.Printf("system %s reset in progress", s.ID)
		return errors.New(fmt.Sprintf("system %s reset in progress", s.ID))
	}

	for _, node := range s.nodeStore {
		node.Reset()
	}

	s.InitEdges()

	log.Printf("starting system %s", s.ID)
	for _, node := range s.nodeStore {
		node.Run(s.ctx, s.wg, s.cancelFunc)
	}

	return nil
}

func (s *System) InitEdges() {
	for _, edge := range s.edgeStore {
		requestChan := make(chan Request, 1000)
		responseChan := make(chan Response, 1000)

		s.nodeStore[edge.SourceID].(Sender).SetOutChannels(requestChan, responseChan)
		s.nodeStore[edge.TargetID].(Receiver).SetInChannels(requestChan, responseChan)
	}
}

func (s *System) AddEdge(senderID string, receiverID string) {
	newEdge := Edge{
		ID:       shortuuid.New(),
		SourceID: senderID,
		TargetID: receiverID,
	}
	s.edgeStore[newEdge.ID] = newEdge
}

type SystemStore map[string]*System

func NewSystemStore() SystemStore {
	return SystemStore{}
}
