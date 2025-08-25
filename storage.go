package unleash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

// Storage is an interface that can be implemented in order to have control over how
// the repository of feature toggles is persisted.
type Storage interface {
	// Init is called to initialize the storage implementation. The backupPath
	// is used to specify the location the data should be stored and the appName
	// can be used in naming.
	Init(backupPath string, appName string)

	// Reset is called after the repository has fetched the feature toggles from the server.
	// If persist is true the implementation of this function should call Persist(). The data
	// passed in here should be owned by the implementer of this interface.
	Reset(data map[string]interface{}, persist bool) error

	// Load is called to load the data from persistent storage and hold it in memory for fast
	// querying.
	Load() error

	// Persist is called when the data in the storage implementation should be persisted to disk.
	Persist() error

	// Get returns the data for the specified feature toggle.
	Get(string) (interface{}, bool)

	// List returns a list of all feature toggles.
	List() []interface{}
}

// DeltaStorage extends Storage with methods for incremental updates
type DeltaStorage interface {
	Storage
	
	// Update adds or updates a single feature
	Update(featureName string, feature interface{}) error
	
	// Delete removes a single feature
	Delete(featureName string) error
	
	// UpdateSegment adds or updates a segment
	UpdateSegment(id int, constraints []api.Constraint) error
	
	// DeleteSegment removes a segment
	DeleteSegment(id int) error
	
	// GetSegments returns all segments
	GetSegments() map[int][]api.Constraint
}

// DefaultStorage is a default Storage implementation.
type DefaultStorage struct {
	appName string
	path    string
	data    map[string]interface{}
}

func (ds *DefaultStorage) Init(backupPath, appName string) {
	ds.appName = appName
	ds.path = filepath.Join(backupPath, fmt.Sprintf("unleash-repo-schema-v1-%s.json", appName))
	ds.data = map[string]interface{}{}
	ds.Load()
}

func (ds *DefaultStorage) Reset(data map[string]interface{}, persist bool) error {
	ds.data = data
	if persist {
		return ds.Persist()
	}
	return nil
}

func (ds *DefaultStorage) Load() error {
	if file, err := os.Open(ds.path); err != nil {
		return err
	} else {
		dec := json.NewDecoder(file)
		var featuresFromFile map[string]api.Feature
		if err := dec.Decode(&featuresFromFile); err != nil {
			return err
		}

		for key, value := range featuresFromFile {
			ds.data[key] = value
		}
	}
	return nil
}

func (ds *DefaultStorage) Persist() error {
	if file, err := os.Create(ds.path); err != nil {
		return err
	} else {
		defer file.Close()
		enc := json.NewEncoder(file)
		if err := enc.Encode(ds.data); err != nil {
			return err
		}
	}
	return nil
}

func (ds DefaultStorage) Get(key string) (interface{}, bool) {
	val, ok := ds.data[key]
	return val, ok
}

func (ds *DefaultStorage) List() []interface{} {
	var features []interface{}
	for _, val := range ds.data {
		features = append(features, val)
	}
	return features
}

// DefaultDeltaStorage extends DefaultStorage with delta support
type DefaultDeltaStorage struct {
	DefaultStorage
	segments map[int][]api.Constraint
	mu       sync.RWMutex
}

// NewDefaultDeltaStorage creates a new DefaultDeltaStorage instance
func NewDefaultDeltaStorage() *DefaultDeltaStorage {
	return &DefaultDeltaStorage{
		DefaultStorage: DefaultStorage{
			data: make(map[string]interface{}),
		},
		segments: make(map[int][]api.Constraint),
	}
}

// Init initializes the storage
func (ds *DefaultDeltaStorage) Init(backupPath, appName string) {
	ds.DefaultStorage.Init(backupPath, appName)
}

// Update adds or updates a single feature
func (ds *DefaultDeltaStorage) Update(featureName string, feature interface{}) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	ds.data[featureName] = feature
	return ds.Persist()
}

// Delete removes a single feature
func (ds *DefaultDeltaStorage) Delete(featureName string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	delete(ds.data, featureName)
	return ds.Persist()
}

// UpdateSegment adds or updates a segment
func (ds *DefaultDeltaStorage) UpdateSegment(id int, constraints []api.Constraint) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	ds.segments[id] = constraints
	return ds.Persist()
}

// DeleteSegment removes a segment
func (ds *DefaultDeltaStorage) DeleteSegment(id int) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	delete(ds.segments, id)
	return ds.Persist()
}

// GetSegments returns all segments
func (ds *DefaultDeltaStorage) GetSegments() map[int][]api.Constraint {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	// Return a copy to prevent concurrent access issues
	segments := make(map[int][]api.Constraint)
	for k, v := range ds.segments {
		segments[k] = v
	}
	return segments
}

// Reset clears and replaces the entire state
func (ds *DefaultDeltaStorage) Reset(data map[string]interface{}, persist bool) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	// Clear existing data
	ds.data = make(map[string]interface{})
	for k, v := range data {
		ds.data[k] = v
	}
	
	// Note: We don't clear segments here to maintain compatibility
	// with existing behavior. This could be revisited if needed.
	
	if persist {
		return ds.Persist()
	}
	return nil
}

// Get returns a feature by name (thread-safe override)
func (ds *DefaultDeltaStorage) Get(key string) (interface{}, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	val, ok := ds.data[key]
	return val, ok
}

// List returns all features (thread-safe override)
func (ds *DefaultDeltaStorage) List() []interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	var features []interface{}
	for _, val := range ds.data {
		features = append(features, val)
	}
	return features
}
