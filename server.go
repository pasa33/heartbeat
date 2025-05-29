package heartbeat

import (
	"fmt"
	"sync"
	"time"
)

type Server struct {
	clients map[string]*BeatPayload
	mu      sync.Mutex
}

func NewServer() *Server {
	return &Server{
		clients: make(map[string]*BeatPayload),
		mu:      sync.Mutex{},
	}
}

func (s *Server) ParseBeat(b BeatPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[b.ClientID] = &b
}

func (s *Server) GetClientStatus(id string) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if val, ok := s.clients[id]; ok {
		return val.Status, nil
	}
	return "", fmt.Errorf("client with id [%s] not found", id)
}

func (s *Server) GetAllClientStatus() []BeatPayload {
	s.mu.Lock()
	defer s.mu.Unlock()

	all := []BeatPayload{}

	for _, v := range s.clients {
		all = append(all, *v)
	}
	return all
}

func (s *Server) AnyError() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, v := range s.clients {
		if v.Status == StatusError {
			return true
		}
	}
	return false
}

func (s *Server) AnyDead() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, v := range s.clients {
		if time.Since(v.Timestamp) > v.BeatDelay {
			return true
		}
	}
	return false
}

func (s *Server) DeleteClient(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, id)
}

func (s *Server) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients = map[string]*BeatPayload{}
}
