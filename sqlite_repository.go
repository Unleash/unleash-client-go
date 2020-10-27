package unleash

import (
	"context"
	"encoding/json"
	"fmt"

	"crawshaw.io/sqlite/sqlitex"
	"github.com/seatgeek/unleash-client-go/v3/internal/api"
)

type sqliteRepository struct {
	handle    *sqlitex.Pool
	callbacks repositoryChannels
}

// NewSqliteRepository Returns a repository setup to read a sqlite database file
func NewSqliteRepository(path string, chans repositoryChannels) (Repository, error) {
	pool, err := sqlitex.Open("file:"+path, 0, 10)
	if err != nil {
		return nil, fmt.Errorf("Could not open features database: %s", err.Error())
	}

	chans.ready <- true

	return &sqliteRepository{handle: pool, callbacks: chans}, nil
}

func (s *sqliteRepository) Close() error {
	return s.handle.Close()
}

func (s *sqliteRepository) GetToggle(key string) *api.Feature {
	conn := s.handle.Get(context.Background())
	if conn == nil {
		s.callbacks.err(fmt.Errorf("Could not get sqlite connection for key %s", key))
		return nil
	}
	defer s.handle.Put(conn)

	stmt, err := conn.Prepare("SELECT strategies, enabled FROM features WHERE full_name = $key")
	if err != nil {
		s.callbacks.err(fmt.Errorf("Could not fetch feature %s: %s", key, err.Error()))
		return nil
	}
	defer stmt.Reset()

	stmt.SetText("$key", key)
	hasRow, err := stmt.Step()
	if err != nil {
		s.callbacks.err(fmt.Errorf("Could not fetch feature %s: %s", key, err.Error()))
		return nil
	}

	if !hasRow {
		return nil
	}

	enabled := stmt.GetInt64("enabled") == 1
	strategies := []api.Strategy{}

	if enabled {
		err = json.Unmarshal([]byte(stmt.GetText("strategies")), &strategies)
		if err != nil {
			s.callbacks.err(fmt.Errorf("Could not decode strategies for feature %s: %s", key, err.Error()))
			return nil
		}
	}

	return &api.Feature{Name: key, Enabled: enabled, Strategies: strategies}
}

func (s *sqliteRepository) GetConfig(key string) string {
	conn := s.handle.Get(context.Background())
	if conn == nil {
		s.callbacks.err(fmt.Errorf("Could not get sqlite connection for key %s", key))
	}
	defer s.handle.Put(conn)

	stmt, err := conn.Prepare("SELECT data FROM runtime_values WHERE full_name = $key")
	if err != nil {
		s.callbacks.err(fmt.Errorf("Could not fetch config value %s: %s", key, err.Error()))
		return "{}"
	}
	defer stmt.Reset()
	
	stmt.SetText("$key", key)
	hasRow, err := stmt.Step()
	if err != nil {
		s.callbacks.err(fmt.Errorf("Could not fetch config value %s: %s", key, err.Error()))
		return "{}"
	}

	if !hasRow {
		s.callbacks.err(fmt.Errorf("Did not find config value %s", key))
		return "{}"
	}

	return stmt.GetText("data")
}
