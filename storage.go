package unleash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Unleash/unleash-client-go/v3/api"
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
