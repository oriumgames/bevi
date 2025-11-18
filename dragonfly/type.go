package dragonfly

import (
	"sync"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/google/uuid"
	"github.com/oriumgames/ark/ecs"
)

// Player wraps a dragonfly *player.Player with an associated ECS entity ID.
// This allows systems to treat runtime players as ECS components/resources;
// the underlying player pointer is embedded for direct API access.
type Player struct {
	*player.Player
	e ecs.Entity
}

// Entity returns the ECS entity associated with this player.
func (p *Player) Entity() ecs.Entity {
	return p.e
}

// Server embeds the underlying dragonfly *server.Server and maintains a
// thread-safe registry mapping player UUIDs to wrapped Player instances.
// Lookups are provided by UUID, name, and XUID for convenience.
type Server struct {
	*server.Server
	mu sync.RWMutex
	p  map[uuid.UUID]*Player
}

// newServer constructs a Server wrapper around an existing *server.Server.
func newServer(srv *server.Server) *Server {
	return &Server{
		Server: srv,
		p:      make(map[uuid.UUID]*Player),
	}
}

// addPlayer registers a Player in the UUID map (internal use during join).
func (srv *Server) addPlayer(p *Player) {
	srv.mu.Lock()
	if srv.p == nil {
		srv.p = make(map[uuid.UUID]*Player)
	}
	srv.p[p.UUID()] = p
	srv.mu.Unlock()
}

// removePlayer removes a Player from the UUID map (internal use during quit).
func (srv *Server) removePlayer(p *Player) {
	srv.mu.Lock()
	delete(srv.p, p.UUID())
	srv.mu.Unlock()
}

// Player looks up a Player by UUID. Returns (nil,false) if not present.
func (srv *Server) Player(uuid uuid.UUID) (*Player, bool) {
	srv.mu.RLock()
	p, ok := srv.p[uuid]
	srv.mu.RUnlock()
	return p, ok
}

// PlayerByName scans registered players for a matching case-sensitive Name().
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

// PlayerByXUID scans registered players for a matching XUID.
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
