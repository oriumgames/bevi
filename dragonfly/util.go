package dragonfly

import (
	"github.com/oriumgames/bevi"
)

func Receive[T PlayerEvent](w *bevi.World, r bevi.EventReader[T], yield func(T) bool) {
	r.ForEach(func(t T) bool {
		if !w.Alive(t.PlayerRef().e) {
			return true
		}
		return yield(t)
	})
}
