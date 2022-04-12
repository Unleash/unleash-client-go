package unleash

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Unleash/unleash-client-go/v3/api"
)

type BootstrapStorage struct {
	backingStore DefaultStorage
	Reader       io.Reader
}

func (bs *BootstrapStorage) Load() error {
	if len(bs.backingStore.data) == 0 && bs.Reader != nil {
		dec := json.NewDecoder(bs.Reader)
		client_features := api.FeatureResponse{}
		if err := dec.Decode(&client_features); err != nil {
			return err
		}
		bs.backingStore.data = client_features.FeatureMap()
	}
	return nil
}

func (bs *BootstrapStorage) Init(backupPath string, appName string) {
	bs.backingStore.Init(backupPath, appName)
	err := bs.Load()

	if err != nil {
		fmt.Printf("Could not load bootstrap storage, because: %s", err.Error())
		return;
	}
}

func (bs *BootstrapStorage) Reset(data map[string]interface{}, persist bool) error {
	return bs.backingStore.Reset(data, persist)
}

func (bs *BootstrapStorage) Persist() error {
	return bs.backingStore.Persist()
}

// Get returns the data for the specified feature toggle.
func (bs *BootstrapStorage) Get(key string) (interface{}, bool) {
	return bs.backingStore.Get(key)
}

func (bs *BootstrapStorage) List() []interface{} {
	return bs.backingStore.List()
}
