package unleash

import (
	"testing"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

// TestRepositorySegmentsFromDeltaStorage tests that repository correctly retrieves segments from DeltaStorage
func TestRepositorySegmentsFromDeltaStorage(t *testing.T) {
	// Create a DeltaStorage with some segments
	deltaStorage := NewDefaultDeltaStorage()
	deltaStorage.Init("/tmp", "test-app")
	
	// Add some segments to storage
	deltaStorage.UpdateSegment(1, []api.Constraint{
		{ContextName: "userId", Operator: "IN", Values: []string{"123", "456"}},
	})
	deltaStorage.UpdateSegment(2, []api.Constraint{
		{ContextName: "environment", Operator: "IN", Values: []string{"prod"}},
	})
	
	// Create repository options with DeltaStorage
	options := repositoryOptions{
		storage:         deltaStorage,
		appName:        "test-app",
		instanceId:     "test-instance",
		refreshInterval: 15000,
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	// Create repository
	repo := newRepository(options, channels)
	defer repo.Close()
	
	// Create a strategy that references segments
	strategy := api.Strategy{
		Name:     "test-strategy",
		Segments: []int{1, 2},
	}
	
	// Test resolveSegmentConstraints
	constraints, err := repo.resolveSegmentConstraints(strategy)
	if err != nil {
		t.Fatalf("Failed to resolve segment constraints: %v", err)
	}
	
	// Should get constraints from both segments
	if len(constraints) != 2 {
		t.Errorf("Expected 2 constraints, got %d", len(constraints))
	}
	
	// Verify the constraints are correct
	foundUserId := false
	foundEnvironment := false
	for _, c := range constraints {
		if c.ContextName == "userId" {
			foundUserId = true
		}
		if c.ContextName == "environment" {
			foundEnvironment = true
		}
	}
	
	if !foundUserId {
		t.Error("userId constraint not found")
	}
	if !foundEnvironment {
		t.Error("environment constraint not found")
	}
}

// TestRepositorySegmentsPollingMode tests that repository still works with regular storage (polling mode)
func TestRepositorySegmentsPollingMode(t *testing.T) {
	// Create regular storage (not DeltaStorage)
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	// Create repository options with regular storage
	options := repositoryOptions{
		storage:         storage,
		appName:        "test-app",
		instanceId:     "test-instance",
		refreshInterval: 15000,
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	// Create repository
	repo := newRepository(options, channels)
	defer repo.Close()
	
	// Manually set segments in repository (simulating what fetch() does)
	repo.Lock()
	repo.segments = map[int][]api.Constraint{
		1: {{ContextName: "userId", Operator: "IN", Values: []string{"789"}}},
		3: {{ContextName: "region", Operator: "IN", Values: []string{"eu"}}},
	}
	repo.Unlock()
	
	// Create a strategy that references segments
	strategy := api.Strategy{
		Name:     "test-strategy",
		Segments: []int{1, 3},
	}
	
	// Test resolveSegmentConstraints
	constraints, err := repo.resolveSegmentConstraints(strategy)
	if err != nil {
		t.Fatalf("Failed to resolve segment constraints: %v", err)
	}
	
	// Should get constraints from both segments
	if len(constraints) != 2 {
		t.Errorf("Expected 2 constraints, got %d", len(constraints))
	}
	
	// Verify the constraints are correct
	foundUserId := false
	foundRegion := false
	for _, c := range constraints {
		if c.ContextName == "userId" {
			foundUserId = true
		}
		if c.ContextName == "region" {
			foundRegion = true
		}
	}
	
	if !foundUserId {
		t.Error("userId constraint not found")
	}
	if !foundRegion {
		t.Error("region constraint not found")
	}
}