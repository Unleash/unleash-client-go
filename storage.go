package unleash_client_go

import (
	"encoding/json"
	"fmt"
	"os"
)

type Storage interface {
	Init(string, string)
	Reset(map[string]interface{}, bool) error
	Load() error
	Persist() error
	Get(string) (interface{}, bool)
}

type defaultStorage struct {
	appName string
	path    string
	data    map[string]interface{}
}

func (ds *defaultStorage) Init(backupPath, appName string) {
	ds.appName = appName
	ds.path = fmt.Sprintf("%sunleash-repo-schema-v1-%s.json", backupPath, appName)
	ds.data = map[string]interface{}{}
	ds.Load()
}

func (ds *defaultStorage) Reset(data map[string]interface{}, persist bool) error {
	ds.data = data
	if persist {
		return ds.Persist()
	}
	return nil
}

func (ds *defaultStorage) Load() error {
	if file, err := os.Open(ds.path); err != nil {
		return err
	} else {
		dec := json.NewDecoder(file)
		if err := dec.Decode(&ds.data); err != nil {
			return err
		}
	}
	return nil
}

func (ds *defaultStorage) Persist() error {
	println(ds.path)
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

func (ds defaultStorage) Get(key string) (interface{}, bool) {
	val, ok := ds.data[key]
	return val, ok
}
