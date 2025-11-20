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
// thread-safe registry mapping player UUIDs to ECS entity IDs.
// Lookups are provided by UUID, name, and XUID for convenience.
type Server struct {
	*server.Server

	mu sync.RWMutex
	pU map[string]ecs.Entity
	pN map[string]ecs.Entity
	pX map[string]ecs.Entity

	mapper *ecs.Map1[Player]
	world  *ecs.World
}

// newServer constructs a Server wrapper around an existing *server.Server.
func newServer(srv *server.Server, world *ecs.World, mapper *ecs.Map1[Player]) *Server {
	return &Server{
		Server: srv,
		pU:     make(map[string]ecs.Entity),
		pN:     make(map[string]ecs.Entity),
		pX:     make(map[string]ecs.Entity),
		mapper: mapper,
		world:  world,
	}
}

// addPlayer registers a Player in the UUID map (internal use during join).
func (srv *Server) addPlayer(p *Player) {
	srv.mu.Lock()
	srv.pU[p.UUID().String()] = p.e
	srv.pN[p.Name()] = p.e
	srv.pX[p.XUID()] = p.e
	srv.mu.Unlock()
}

// removePlayer removes a Player from the UUID map (internal use during quit).
func (srv *Server) removePlayer(p *Player) {
	srv.mu.Lock()
	delete(srv.pU, p.UUID().String())
	delete(srv.pN, p.Name())
	delete(srv.pX, p.XUID())
	srv.mu.Unlock()
}

func (srv *Server) PlayerEntity(uuid uuid.UUID) (ecs.Entity, bool) {
	srv.mu.RLock()
	p, ok := srv.pU[uuid.String()]
	srv.mu.RUnlock()
	return p, ok
}

func (srv *Server) PlayerEntityByName(name string) (ecs.Entity, bool) {
	srv.mu.RLock()
	p, ok := srv.pN[name]
	srv.mu.RUnlock()
	return p, ok
}

func (srv *Server) PlayerEntityByXUID(xuid string) (ecs.Entity, bool) {
	srv.mu.RLock()
	p, ok := srv.pX[xuid]
	srv.mu.RUnlock()
	return p, ok
}

func (srv *Server) Player(e ecs.Entity) (*Player, bool) {
	if !srv.world.Alive(e) {
		return nil, false
	}
	p := srv.mapper.Get(e)
	return p, (p != nil)
}

func (srv *Server) PlayerByName(name string) (*Player, bool) {
	e, ok := srv.PlayerEntityByName(name)
	if !ok {
		return nil, false
	}
	if !srv.world.Alive(e) {
		return nil, false
	}
	p := srv.mapper.Get(e)
	return p, (p != nil)
}

func (srv *Server) PlayerByUUID(uuid uuid.UUID) (*Player, bool) {
	e, ok := srv.PlayerEntity(uuid)
	if !ok {
		return nil, false
	}
	if !srv.world.Alive(e) {
		return nil, false
	}
	p := srv.mapper.Get(e)
	return p, (p != nil)
}

func (srv *Server) PlayerByXUID(xuid string) (*Player, bool) {
	e, ok := srv.PlayerEntityByXUID(xuid)
	if !ok {
		return nil, false
	}
	if !srv.world.Alive(e) {
		return nil, false
	}
	p := srv.mapper.Get(e)
	return p, (p != nil)
}
