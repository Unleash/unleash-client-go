package unleash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	Reset(data map[string]*api.Feature, persist bool) error

	// Load is called to load the data from persistent storage and hold it in memory for fast
	// querying.
	Load() error

	// Persist is called when the data in the storage implementation should be persisted to disk.
	Persist() error

	// Get returns the data for the specified feature toggle.
	Get(string) (*api.Feature, bool)

	// List returns a list of all feature toggles.
	List() []*api.Feature
}

// DefaultStorage is a default Storage implementation.
type DefaultStorage struct {
	appName string
	path    string
	data    map[string]*api.Feature
}

func (ds *DefaultStorage) Init(backupPath, appName string) {
	ds.appName = appName
	ds.path = filepath.Join(backupPath, fmt.Sprintf("unleash-repo-schema-v1-%s.json", appName))
	ds.data = make(map[string]*api.Feature)
	ds.Load()
}

func (ds *DefaultStorage) Reset(data map[string]*api.Feature, persist bool) error {
	ds.data = data
	if persist {
		return ds.Persist()
	}
	return nil
}

func (ds *DefaultStorage) Load() error {
	file, err := os.Open(ds.path)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	var featuresFromFile map[string]api.Feature
	if err := dec.Decode(&featuresFromFile); err != nil {
		return err
	}

	for key, value := range featuresFromFile {
		feat := value // copy into a new variable to take address safely
		ds.data[key] = &feat
	}

	return nil
}

func (ds *DefaultStorage) Persist() error {
	file, err := os.Create(ds.path)
	if err != nil {
		return err
	}
	defer file.Close()

	out := make(map[string]api.Feature, len(ds.data))
	for k, v := range ds.data {
		if v != nil {
			out[k] = *v
		}
	}

	enc := json.NewEncoder(file)
	if err := enc.Encode(out); err != nil {
		return err
	}
	return nil
}

func (ds *DefaultStorage) Get(key string) (*api.Feature, bool) {
	val, ok := ds.data[key]
	return val, ok
}

func (ds *DefaultStorage) List() []*api.Feature {
	features := make([]*api.Feature, 0, len(ds.data))
	for _, val := range ds.data {
		features = append(features, val)
	}
	return features
}
