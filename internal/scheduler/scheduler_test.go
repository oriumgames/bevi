package scheduler

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestTopologicalSort(t *testing.T) {
	s := NewScheduler()

	// Create systems with conflicting access to force sequential execution
	intType := reflect.TypeOf(0)

	sys1 := &System{
		Name:  "sys1",
		Stage: 0,
		Meta: SystemMeta{
			Access: AccessMeta{Writes: []reflect.Type{intType}},
		},
	}
	sys2 := &System{
		Name:  "sys2",
		Stage: 0,
		Meta: SystemMeta{
			Access: AccessMeta{Writes: []reflect.Type{intType}},
			After:  []string{"sys1"},
		},
	}
	sys3 := &System{
		Name:  "sys3",
		Stage: 0,
		Meta: SystemMeta{
			Access: AccessMeta{Writes: []reflect.Type{intType}},
			After:  []string{"sys2"},
		},
	}

	s.AddSystem(sys1)
	s.AddSystem(sys2)
	s.AddSystem(sys3)

	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	batches := s.batches[0]
	if len(batches) != 3 {
		t.Errorf("Expected 3 batches (sequential), got %d", len(batches))
	}
}

func TestParallelBatching(t *testing.T) {
	s := NewScheduler()

	// Create systems with non-conflicting access
	intType := reflect.TypeOf(0)
	stringType := reflect.TypeOf("")

	sys1 := &System{
		Name:  "sys1",
		Stage: 0,
		Meta: SystemMeta{
			Access: AccessMeta{
				Reads: []reflect.Type{intType},
			},
		},
	}

	sys2 := &System{
		Name:  "sys2",
		Stage: 0,
		Meta: SystemMeta{
			Access: AccessMeta{
				Reads: []reflect.Type{stringType},
			},
		},
	}

	s.AddSystem(sys1)
	s.AddSystem(sys2)

	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	batches := s.batches[0]
	if len(batches) != 1 {
		t.Errorf("Expected 1 batch (parallel), got %d", len(batches))
	}

	if len(batches[0]) != 2 {
		t.Errorf("Expected 2 systems in batch, got %d", len(batches[0]))
	}
}

func TestAccessConflicts(t *testing.T) {
	intType := reflect.TypeOf(0)

	tests := []struct {
		name      string
		access1   AccessMeta
		access2   AccessMeta
		conflicts bool
	}{
		{
			name:      "read-read no conflict",
			access1:   AccessMeta{Reads: []reflect.Type{intType}},
			access2:   AccessMeta{Reads: []reflect.Type{intType}},
			conflicts: false,
		},
		{
			name:      "write-read conflict",
			access1:   AccessMeta{Writes: []reflect.Type{intType}},
			access2:   AccessMeta{Reads: []reflect.Type{intType}},
			conflicts: true,
		},
		{
			name:      "write-write conflict",
			access1:   AccessMeta{Writes: []reflect.Type{intType}},
			access2:   AccessMeta{Writes: []reflect.Type{intType}},
			conflicts: true,
		},
		{
			name:      "different types no conflict",
			access1:   AccessMeta{Writes: []reflect.Type{intType}},
			access2:   AccessMeta{Writes: []reflect.Type{reflect.TypeOf("")}},
			conflicts: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.access1.Conflicts(tt.access2)
			if result != tt.conflicts {
				t.Errorf("Expected conflicts=%v, got %v", tt.conflicts, result)
			}
		})
	}
}

func TestEveryGating(t *testing.T) {
	sys := &System{
		Name: "test",
		Meta: SystemMeta{
			Every: 100 * time.Millisecond,
		},
		LastRun: time.Now(),
	}

	// Should not run immediately
	if sys.ShouldRun(time.Now()) {
		t.Error("System should not run immediately after last run")
	}

	// Should run after interval
	future := time.Now().Add(101 * time.Millisecond)
	if !sys.ShouldRun(future) {
		t.Error("System should run after interval")
	}
}

func TestCyclicDependency(t *testing.T) {
	s := NewScheduler()

	sys1 := &System{
		Name:  "sys1",
		Stage: 0,
		Meta:  SystemMeta{After: []string{"sys2"}},
	}
	sys2 := &System{
		Name:  "sys2",
		Stage: 0,
		Meta:  SystemMeta{After: []string{"sys1"}},
	}

	s.AddSystem(sys1)
	s.AddSystem(sys2)

	if err := s.Build(); err == nil {
		t.Error("Expected error for cyclic dependency")
	}
}

func TestSetDependencies(t *testing.T) {
	s := NewScheduler()

	sys1 := &System{Name: "sys1", Stage: 0, Meta: SystemMeta{Set: "GroupA"}}
	sys2 := &System{Name: "sys2", Stage: 0, Meta: SystemMeta{Set: "GroupA"}}
	sys3 := &System{Name: "sys3", Stage: 0, Meta: SystemMeta{After: []string{"GroupA"}}}

	s.AddSystem(sys1)
	s.AddSystem(sys2)
	s.AddSystem(sys3)

	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// sys3 should come after both sys1 and sys2
	batches := s.batches[0]
	lastBatch := batches[len(batches)-1]

	found := false
	for _, sys := range lastBatch {
		if sys.Name == "sys3" {
			found = true
			break
		}
	}

	if !found {
		t.Error("sys3 should be in the last batch")
	}
}

func TestSystemExecution(t *testing.T) {
	s := NewScheduler()

	executed := false
	sys := &System{
		Name:  "test",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			executed = true
		},
	}

	s.AddSystem(sys)
	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	s.RunStage(context.Background(), 0, nil, nil)

	if !executed {
		t.Error("System was not executed")
	}
}
