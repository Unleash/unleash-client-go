package unleash

import (
	"encoding/json"
	"io"

	"github.com/Unleash/unleash-go-sdk/v6/api"
)

type BootstrapStorage struct {
	backingStore DefaultStorage
	Reader       io.Reader
}

func (ds *BootstrapStorage) Init(backupPath, appName string) {
	ds.backingStore.Init(backupPath, appName)
}

func (bs *BootstrapStorage) Load() (*api.FeatureResponse, error) {
	if bs.Reader == nil {
		return nil, nil
	}

	dec := json.NewDecoder(bs.Reader)
	clientFeatures := api.FeatureResponse{}
	if err := dec.Decode(&clientFeatures); err == nil {
		return &clientFeatures, nil
	}

	// If we reach here, there was an error decoding the features from the reader
	// So we fall back to loading from the decorated store. If that also fails
	// it's not a major issue since the SDK will hydrate from the API
	if data, err := bs.backingStore.Load(); err == nil {
		return data, nil
	} else {
		return nil, err
	}
}

func (bs *BootstrapStorage) Persist(features *api.FeatureResponse) error {
	return bs.backingStore.Persist(features)
}
