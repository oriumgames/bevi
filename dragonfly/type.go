package dragonfly

import (
	"sync"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/google/uuid"
	"github.com/mlange-42/ark/ecs"
)

type Player struct {
	*player.Player
	e ecs.Entity
}

func (p *Player) Entity() ecs.Entity {
	return p.e
}

type Server struct {
	*server.Server
	mu sync.RWMutex
	p  map[uuid.UUID]*Player
}

func newServer(srv *server.Server) *Server {
	return &Server{
		Server: srv,
		p:      make(map[uuid.UUID]*Player),
	}
}

func (srv *Server) addPlayer(p *Player) {
	srv.mu.Lock()
	if srv.p == nil {
		srv.p = make(map[uuid.UUID]*Player)
	}
	srv.p[p.UUID()] = p
	srv.mu.Unlock()
}

func (srv *Server) removePlayer(p *Player) {
	srv.mu.Lock()
	delete(srv.p, p.UUID())
	srv.mu.Unlock()
}

func (srv *Server) Player(uuid uuid.UUID) (*Player, bool) {
	srv.mu.RLock()
	p, ok := srv.p[uuid]
	srv.mu.RUnlock()
	return p, ok
}

func (srv *Server) PlayerByName(name string) (*Player, bool) {
	srv.mu.RLock()
	for _, p := range srv.p {
		if p.Name() == name {
			srv.mu.RUnlock()
			return p, true
		}
	}
	srv.mu.RUnlock()
	return nil, false
}

func (srv *Server) PlayerByXUID(xuid string) (*Player, bool) {
	srv.mu.RLock()
	for _, p := range srv.p {
		if p.XUID() == xuid {
			srv.mu.RUnlock()
			return p, true
		}
	}
	srv.mu.RUnlock()
	return nil, false
}
