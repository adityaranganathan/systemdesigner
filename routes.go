package main

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

func RegisterRoutes() *mux.Router {
	systemStore := NewSystemStore()
	router := mux.NewRouter()

	router.Methods(http.MethodPost).Path("/api/systems").HandlerFunc(getCreateSystemHandler(systemStore))
	router.Methods(http.MethodGet).Path("/api/systems/{systemID}").HandlerFunc(getSystemHandler(systemStore))
	router.Methods(http.MethodPut).Path("/api/systems/{systemID}/start").HandlerFunc(getStartSystemHandler(systemStore))
	router.Methods(http.MethodGet).Path("/api/systems/{systemID}/metrics").HandlerFunc(getSystemMetricsHandler(systemStore))

	router.Methods(http.MethodPost).Path("/api/systems/{systemID}/nodes").HandlerFunc(getCreateNodeHandler(systemStore))
	router.Methods(http.MethodPatch).Path("/api/systems/{systemID}/nodes/{nodeID}").HandlerFunc(getUpdateNodeHandler(systemStore))
	router.Methods(http.MethodDelete).Path("/api/systems/{systemID}/nodes").HandlerFunc(getDeleteNodesHandler(systemStore))

	router.Methods(http.MethodPost).Path("/api/systems/{systemID}/edges").HandlerFunc(getCreateEdgeHandler(systemStore))
	router.Methods(http.MethodDelete).Path("/api/systems/{systemID}/edges").HandlerFunc(getDeleteEdgesHandler(systemStore))

	return router
}

type CreateSystemResponse struct {
	ID string `json:"id"`
}

func getCreateSystemHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {

		system := NewSystem()
		system = DefaultSystem(system)

		systemStore[system.ID] = system

		writer.WriteHeader(http.StatusCreated)
		err := json.NewEncoder(writer).Encode(CreateSystemResponse{ID: system.ID})
		if err != nil {
			encodeError(writer, err, http.StatusInternalServerError)
			return
		}
	}
}

type GetSystemResponse struct {
	ID    string         `json:"id"`
	Nodes []NodeResponse `json:"nodes"`
	Edges []EdgeResponse `json:"edges"`
}

type NodeResponse struct {
	ID       string       `json:"id"`
	Position Position     `json:"position"`
	Data     DataResponse `json:"data"`
}

type DataResponse struct {
	Type    string   `json:"type"`
	Metrics []Metric `json:"metrics"`
}

type EdgeResponse struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func getSystemHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {

		vars := mux.Vars(request)

		system, ok := systemStore[vars["systemID"]]
		if !ok {
			encodeError(writer, errors.New("system not found"), http.StatusNotFound)
			return
		}

		response := GetSystemResponse{
			ID:    system.ID,
			Nodes: []NodeResponse{},
			Edges: []EdgeResponse{},
		}
		for _, node := range system.nodeStore {
			response.Nodes = append(response.Nodes, NodeResponse{
				ID:       node.GetID(),
				Position: node.GetPosition(),
				Data:     DataResponse{Type: node.GetType(), Metrics: node.GetMetrics()},
			})
		}
		for _, edge := range system.edgeStore {
			response.Edges = append(response.Edges, EdgeResponse{
				ID:     edge.ID,
				Source: edge.SourceID,
				Target: edge.TargetID,
			})
		}

		err := json.NewEncoder(writer).Encode(response)
		if err != nil {
			encodeError(writer, err, http.StatusInternalServerError)
			return
		}
	}
}

func getStartSystemHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)

		system := systemStore[vars["systemID"]]
		err := system.Start()
		if err != nil {
			encodeError(writer, err, http.StatusInternalServerError)
			return
		}

		writer.WriteHeader(http.StatusOK)
	}
}

func getSystemMetricsHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		vars := mux.Vars(request)

		system, exists := systemStore[vars["systemID"]]
		if !exists {
			encodeError(writer, errors.New("system not found"), http.StatusNotFound)
			return
		}
		messages := system.messages

		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			encodeError(writer, err, http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					log.Print("read error: ", err)
					return
				}
			}
		}()

		for {
			select {
			case <-done:
				return
			case msg := <-messages:

				msgBytes, err := json.Marshal(msg)
				if err != nil {
					log.Print("marshall error: ", err)
					continue
				}

				err = conn.WriteMessage(websocket.TextMessage, msgBytes)
				if err != nil {
					log.Print("write error: ", err)
					continue
				}
			}
		}
	}
}

type CreateNodeRequest struct {
	Type string `json:"type"`
}

func getCreateNodeHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)

		var body CreateNodeRequest
		err := json.NewDecoder(request.Body).Decode(&body)
		if err != nil {
			encodeError(writer, err, http.StatusInternalServerError)
			return
		}

		system := systemStore[vars["systemID"]]
		newNode := system.AddNode(body.Type)

		writer.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(writer).Encode(newNode)
		if err != nil {
			encodeError(writer, err, http.StatusInternalServerError)
			return
		}
	}
}

type UpdateNodeRequest struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func getUpdateNodeHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)

		var body UpdateNodeRequest
		err := json.NewDecoder(request.Body).Decode(&body)
		if err != nil {
			encodeError(writer, err, http.StatusInternalServerError)
			return
		}

		system := systemStore[vars["systemID"]]
		system.nodeStore[vars["nodeID"]].SetPosition(Position{
			X: int(body.X),
			Y: int(body.Y),
		})

		writer.WriteHeader(http.StatusOK)
	}
}

func getDeleteNodesHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)

		system := systemStore[vars["systemID"]]
		for _, nodeID := range request.URL.Query()["id"] {
			log.Printf("deleting system %s node %s", system.ID, nodeID)
			delete(system.nodeStore, nodeID)
		}

		writer.WriteHeader(http.StatusOK)
	}
}

type CreateConnectionRequest struct {
	SourceID string `json:"source"`
	TargetID string `json:"target"`
}

func getCreateEdgeHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)

		var body CreateConnectionRequest
		err := json.NewDecoder(request.Body).Decode(&body)
		if err != nil {
			encodeError(writer, err, http.StatusInternalServerError)
			return
		}

		system := systemStore[vars["systemID"]]

		system.AddEdge(body.SourceID, body.TargetID)

		writer.WriteHeader(http.StatusCreated)
	}
}

func getDeleteEdgesHandler(systemStore SystemStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)

		system := systemStore[vars["systemID"]]
		for _, edgeID := range request.URL.Query()["id"] {
			log.Printf("deleting system %s edge %s", system.ID, edgeID)
			delete(system.edgeStore, edgeID)
		}

		writer.WriteHeader(http.StatusOK)
	}
}

func encodeError(writer http.ResponseWriter, err error, code int) {
	log.Print(err)

	http.Error(writer, err.Error(), code)
}
