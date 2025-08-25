package unleash

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

var errPersist = errors.New("persist error")

// MockDeltaStorage implements DeltaStorage for testing
type MockDeltaStorage struct {
	data            map[string]interface{}
	segments        map[int][]api.Constraint
	mu              sync.RWMutex
	persistCalled   int
	persistCallsMu  sync.Mutex
	failPersist     bool
	updateCallCount int
	deleteCallCount int
}

func NewMockDeltaStorage() *MockDeltaStorage {
	return &MockDeltaStorage{
		data:     make(map[string]interface{}),
		segments: make(map[int][]api.Constraint),
	}
}

// Storage interface methods
func (m *MockDeltaStorage) Init(backupPath string, appName string) {
	// No-op for testing
}

func (m *MockDeltaStorage) Reset(data map[string]interface{}, persist bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.data = make(map[string]interface{})
	for k, v := range data {
		m.data[k] = v
	}
	
	if persist {
		return m.Persist()
	}
	return nil
}

func (m *MockDeltaStorage) Load() error {
	return nil
}

func (m *MockDeltaStorage) Persist() error {
	m.persistCallsMu.Lock()
	defer m.persistCallsMu.Unlock()
	
	m.persistCalled++
	if m.failPersist {
		return errPersist
	}
	return nil
}

func (m *MockDeltaStorage) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	val, ok := m.data[key]
	return val, ok
}

func (m *MockDeltaStorage) List() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var features []interface{}
	for _, val := range m.data {
		features = append(features, val)
	}
	return features
}

// DeltaStorage interface methods
func (m *MockDeltaStorage) Update(featureName string, feature interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.updateCallCount++
	m.data[featureName] = feature
	return m.Persist()
}

func (m *MockDeltaStorage) Delete(featureName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.deleteCallCount++
	delete(m.data, featureName)
	return m.Persist()
}

func (m *MockDeltaStorage) UpdateSegment(id int, constraints []api.Constraint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.segments[id] = constraints
	return m.Persist()
}

func (m *MockDeltaStorage) DeleteSegment(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.segments, id)
	return m.Persist()
}

func (m *MockDeltaStorage) GetSegments() map[int][]api.Constraint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent concurrent map access
	segments := make(map[int][]api.Constraint)
	for k, v := range m.segments {
		segments[k] = v
	}
	return segments
}

func (m *MockDeltaStorage) GetPersistCallCount() int {
	m.persistCallsMu.Lock()
	defer m.persistCallsMu.Unlock()
	return m.persistCalled
}

// Test DeltaStorage operations
func TestDeltaStorageUpdate(t *testing.T) {
	storage := NewMockDeltaStorage()
	
	feature := api.Feature{
		Name:    "test-feature",
		Enabled: true,
	}
	
	// Test Update adds new feature
	err := storage.Update("test-feature", feature)
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
	
	// Verify feature was added
	stored, exists := storage.Get("test-feature")
	if !exists {
		t.Error("Feature should exist after Update")
	}
	
	if f, ok := stored.(api.Feature); ok {
		if f.Name != "test-feature" || !f.Enabled {
			t.Error("Stored feature doesn't match")
		}
	} else {
		t.Error("Stored value is not a Feature")
	}
	
	// Verify persist was called
	if storage.GetPersistCallCount() != 1 {
		t.Errorf("Persist called %d times, want 1", storage.GetPersistCallCount())
	}
	
	// Test Update modifies existing feature
	feature.Enabled = false
	err = storage.Update("test-feature", feature)
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
	
	stored, _ = storage.Get("test-feature")
	if f, ok := stored.(api.Feature); ok {
		if f.Enabled {
			t.Error("Feature should be disabled after update")
		}
	}
	
	if storage.GetPersistCallCount() != 2 {
		t.Errorf("Persist called %d times, want 2", storage.GetPersistCallCount())
	}
}

func TestDeltaStorageDelete(t *testing.T) {
	storage := NewMockDeltaStorage()
	
	// Add a feature
	feature := api.Feature{Name: "delete-me", Enabled: true}
	storage.Update("delete-me", feature)
	
	// Verify it exists
	if _, exists := storage.Get("delete-me"); !exists {
		t.Fatal("Feature should exist before delete")
	}
	
	// Delete the feature
	err := storage.Delete("delete-me")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
	
	// Verify it's gone
	if _, exists := storage.Get("delete-me"); exists {
		t.Error("Feature should not exist after delete")
	}
	
	// Verify persist was called (once for add, once for delete)
	if storage.GetPersistCallCount() != 2 {
		t.Errorf("Persist called %d times, want 2", storage.GetPersistCallCount())
	}
	
	// Delete non-existent feature should not error
	err = storage.Delete("non-existent")
	if err != nil {
		t.Errorf("Delete() non-existent error = %v", err)
	}
}

func TestDeltaStorageSegments(t *testing.T) {
	storage := NewMockDeltaStorage()
	
	constraints := []api.Constraint{
		{
			ContextName: "userId",
			Operator:    "IN",
			Values:      []string{"123", "456"},
		},
	}
	
	// Test UpdateSegment
	err := storage.UpdateSegment(1, constraints)
	if err != nil {
		t.Errorf("UpdateSegment() error = %v", err)
	}
	
	segments := storage.GetSegments()
	if len(segments) != 1 {
		t.Errorf("Expected 1 segment, got %d", len(segments))
	}
	
	if len(segments[1]) != 1 {
		t.Error("Segment constraints not stored correctly")
	}
	
	// Test updating existing segment
	newConstraints := []api.Constraint{
		{
			ContextName: "environment",
			Operator:    "IN",
			Values:      []string{"prod"},
		},
	}
	
	err = storage.UpdateSegment(1, newConstraints)
	if err != nil {
		t.Errorf("UpdateSegment() error = %v", err)
	}
	
	segments = storage.GetSegments()
	if len(segments[1]) != 1 || segments[1][0].ContextName != "environment" {
		t.Error("Segment should be updated")
	}
	
	// Test DeleteSegment
	err = storage.DeleteSegment(1)
	if err != nil {
		t.Errorf("DeleteSegment() error = %v", err)
	}
	
	segments = storage.GetSegments()
	if len(segments) != 0 {
		t.Errorf("Expected 0 segments after delete, got %d", len(segments))
	}
	
	// Verify persist was called for each operation
	expectedPersistCalls := 3 // UpdateSegment, UpdateSegment, DeleteSegment
	if storage.GetPersistCallCount() != expectedPersistCalls {
		t.Errorf("Persist called %d times, want %d", storage.GetPersistCallCount(), expectedPersistCalls)
	}
}

func TestDeltaStorageConcurrency(t *testing.T) {
	storage := NewMockDeltaStorage()
	
	// Number of concurrent operations
	numOps := 100
	var wg sync.WaitGroup
	wg.Add(numOps * 3) // 3 types of operations
	
	// Concurrent updates
	for i := 0; i < numOps; i++ {
		go func(idx int) {
			defer wg.Done()
			featureName := fmt.Sprintf("feature-%d", idx)
			feature := api.Feature{Name: featureName, Enabled: true}
			storage.Update(featureName, feature)
		}(i)
	}
	
	// Concurrent deletes
	for i := 0; i < numOps; i++ {
		go func(idx int) {
			defer wg.Done()
			featureName := fmt.Sprintf("feature-%d", idx)
			storage.Delete(featureName)
		}(i)
	}
	
	// Concurrent reads
	for i := 0; i < numOps; i++ {
		go func(idx int) {
			defer wg.Done()
			featureName := fmt.Sprintf("feature-%d", idx)
			storage.Get(featureName)
			storage.List()
			storage.GetSegments()
		}(i)
	}
	
	// Wait for all operations to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()
	
	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent operations timed out - possible deadlock")
	}
}

func TestDeltaStorageTypeAssertion(t *testing.T) {
	t.Run("DeltaStorage implementation", func(t *testing.T) {
		storage := NewMockDeltaStorage()
		
		// Test that MockDeltaStorage implements Storage
		var _ Storage = storage
		
		// Test that MockDeltaStorage implements DeltaStorage
		var _ DeltaStorage = storage
		
		// Test type assertion
		if ds, ok := interface{}(storage).(DeltaStorage); !ok {
			t.Error("MockDeltaStorage should implement DeltaStorage")
		} else {
			// Should be able to use delta methods
			ds.Update("test", api.Feature{Name: "test"})
			if storage.updateCallCount != 1 {
				t.Error("Update should have been called")
			}
		}
	})
	
	t.Run("Non-DeltaStorage fallback", func(t *testing.T) {
		// Use regular DefaultStorage which doesn't implement DeltaStorage
		storage := &DefaultStorage{}
		storage.Init("", "test-app")
		
		// Type assertion should fail
		if _, ok := interface{}(storage).(DeltaStorage); ok {
			t.Error("DefaultStorage should not implement DeltaStorage")
		}
		
		// Should still work as regular Storage
		var _ Storage = storage
		storage.Reset(map[string]interface{}{"test": api.Feature{Name: "test"}}, false)
		
		if _, exists := storage.Get("test"); !exists {
			t.Error("Regular storage operations should still work")
		}
	})
}

func TestDeltaStoragePersistErrors(t *testing.T) {
	storage := NewMockDeltaStorage()
	storage.failPersist = true
	
	// Update should return persist error
	err := storage.Update("test", api.Feature{Name: "test"})
	if err == nil {
		t.Error("Expected persist error from Update")
	}
	
	// Delete should return persist error
	err = storage.Delete("test")
	if err == nil {
		t.Error("Expected persist error from Delete")
	}
	
	// UpdateSegment should return persist error
	err = storage.UpdateSegment(1, []api.Constraint{})
	if err == nil {
		t.Error("Expected persist error from UpdateSegment")
	}
	
	// DeleteSegment should return persist error
	err = storage.DeleteSegment(1)
	if err == nil {
		t.Error("Expected persist error from DeleteSegment")
	}
}

func TestDeltaStorageReset(t *testing.T) {
	storage := NewMockDeltaStorage()
	
	// Add some initial data
	storage.Update("feature1", api.Feature{Name: "feature1"})
	storage.Update("feature2", api.Feature{Name: "feature2"})
	storage.UpdateSegment(1, []api.Constraint{{ContextName: "test"}})
	
	// Reset should clear and replace data
	newData := map[string]interface{}{
		"feature3": api.Feature{Name: "feature3"},
	}
	
	err := storage.Reset(newData, true)
	if err != nil {
		t.Errorf("Reset() error = %v", err)
	}
	
	// Old features should be gone
	if _, exists := storage.Get("feature1"); exists {
		t.Error("feature1 should not exist after Reset")
	}
	if _, exists := storage.Get("feature2"); exists {
		t.Error("feature2 should not exist after Reset")
	}
	
	// New feature should exist
	if _, exists := storage.Get("feature3"); !exists {
		t.Error("feature3 should exist after Reset")
	}
	
	// Note: Reset doesn't clear segments in current implementation
	// This behavior should be documented or fixed
}

func TestDefaultDeltaStorage(t *testing.T) {
	// Test that we can create a DefaultDeltaStorage that extends DefaultStorage
	storage := &DefaultDeltaStorage{
		DefaultStorage: DefaultStorage{
			data: make(map[string]interface{}),
		},
		segments: make(map[int][]api.Constraint),
	}
	
	storage.Init("/tmp", "test-app")
	
	// Test Update
	err := storage.Update("test-feature", api.Feature{Name: "test-feature", Enabled: true})
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
	
	// Test Get
	if feature, exists := storage.Get("test-feature"); !exists {
		t.Error("Feature should exist after Update")
	} else if f, ok := feature.(api.Feature); !ok || f.Name != "test-feature" {
		t.Error("Retrieved feature doesn't match")
	}
	
	// Test Delete
	err = storage.Delete("test-feature")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
	
	if _, exists := storage.Get("test-feature"); exists {
		t.Error("Feature should not exist after Delete")
	}
	
	// Test segment operations
	constraints := []api.Constraint{{ContextName: "userId", Operator: "IN"}}
	err = storage.UpdateSegment(1, constraints)
	if err != nil {
		t.Errorf("UpdateSegment() error = %v", err)
	}
	
	segments := storage.GetSegments()
	if len(segments) != 1 {
		t.Errorf("Expected 1 segment, got %d", len(segments))
	}
	
	err = storage.DeleteSegment(1)
	if err != nil {
		t.Errorf("DeleteSegment() error = %v", err)
	}
	
	segments = storage.GetSegments()
	if len(segments) != 0 {
		t.Errorf("Expected 0 segments after delete, got %d", len(segments))
	}
}