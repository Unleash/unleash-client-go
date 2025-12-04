package unleash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Unleash/unleash-go-sdk/v6/api"
)

// Storage controls persistence of the SDK state (features + segments).
type Storage interface {
	// Init is called to initialize the storage implementation. The backupPath
	// is used to specify the location the data should be stored and the appName
	// can be used in naming.
	Init(backupPath string, appName string)
	// Load retrieves data from the backing store but does not cache
	Load() (*api.FeatureResponse, error)
	// Persist is called when data in the storage implementation should be persisted to the backing data store
	Persist(state *api.FeatureResponse) error
}

type DefaultStorage struct {
	path string
}

func (ds *DefaultStorage) Init(backupPath, appName string) {
	ds.path = filepath.Join(backupPath, fmt.Sprintf("unleash-repo-schema-v1-%s.json", appName))
}

func (ds *DefaultStorage) Load() (*api.FeatureResponse, error) {
	file, err := os.Open(ds.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	var state api.FeatureResponse
	if err := dec.Decode(&state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (ds *DefaultStorage) Persist(state *api.FeatureResponse) error {
	file, err := os.Create(ds.path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	if err := enc.Encode(state); err != nil {
		return err
	}
	return nil
}
