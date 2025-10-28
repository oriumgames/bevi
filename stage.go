package bevi

// Stage represents a scheduling stage in the ECS execution pipeline.
type Stage int

const (
	// Startup runs once at application initialization.
	Startup Stage = iota
	// Update runs every frame for game logic.
	Update
)

// String returns the string representation of a stage.
func (s Stage) String() string {
	switch s {
	case Startup:
		return "Startup"
	case Update:
		return "Update"
	default:
		return "Unknown"
	}
}
