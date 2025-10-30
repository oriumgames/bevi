package bevi

// Stage represents a scheduling stage in the ECS execution pipeline.
type Stage int

const (
	// PreStartup runs once before the main Startup stage.
	PreStartup Stage = iota
	// Startup runs once at application initialization.
	Startup
	// PostStartup runs once after Startup for early initialization finalization.
	PostStartup
	// PreUpdate runs once before the main Update stage for preparatory systems.
	PreUpdate
	// Update runs every frame for game logic.
	Update
	// PostUpdate runs once after the main Update stage for cleanup or finalization.
	PostUpdate
)

// String returns the string representation of a stage.
func (s Stage) String() string {
	switch s {
	case PreStartup:
		return "PreStartup"
	case Startup:
		return "Startup"
	case PostStartup:
		return "PostStartup"
	case PreUpdate:
		return "PreUpdate"
	case Update:
		return "Update"
	case PostUpdate:
		return "PostUpdate"
	default:
		return "Unknown"
	}
}
