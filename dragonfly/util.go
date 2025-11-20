package dragonfly

import (
	"fmt"

	"github.com/oriumgames/ark/ecs"
	"github.com/oriumgames/bevi"
)

func Receive[T PlayerEvent](srv ecs.Resource[Server], r bevi.EventReader[T], yield func(T, *Player) bool) {
	r.ForEach(func(t T) bool {
		fmt.Println(t)
		p, ok := srv.Get().Player(t.Player())
		if !ok {
			return true
		}
		return yield(t, p)
	})
}
