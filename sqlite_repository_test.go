package unleash

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"crawshaw.io/sqlite/sqlitex"
	"github.com/stretchr/testify/assert"
)

func buildTestSqliteRepository(assert *assert.Assertions) (Repository, errorChannels, string) {
	errChannels := errorChannels{
		errors:   make(chan error, 1000),
		warnings: make(chan error, 1000),
	}
	ready := make(chan bool, 1)
	chans := repositoryChannels{
		errorChannels: errChannels,
		ready:         ready,
	}

	tempFile, err := ioutil.TempFile(os.TempDir(), ".sqlite")
	assert.NoError(err)
	defer tempFile.Close()

	path := tempFile.Name()
	contents, _ := ioutil.ReadFile("fixtures/features.sql")
	assert.NotNil(contents)

	pool, err := sqlitex.Open("file:"+path, 0, 1)
	assert.NoError(err)

	conn := pool.Get(context.Background())
	defer pool.Put(conn)

	err = sqlitex.ExecScript(conn, string(contents))
	assert.NoError(err)

	repository, err := NewSqliteRepository(path, chans)
	assert.NoError(err)

	return repository, errChannels, path
}

// TestSqliteRepository_GetFeature
func TestSqliteRepository_GetFeatureSingle(t *testing.T) {
	assert := assert.New(t)

	repository, errChannels, _ := buildTestSqliteRepository(assert)
	defer repository.Close()
	f := repository.GetToggle("dummy.feature1")

	select {
	case err := <-errChannels.errors:
		assert.NoError(err)
		return
	case err := <-errChannels.warnings:
		assert.NoError(err)
		return
	default:
	}

	assert.Equal(f.Name, "dummy.feature1")
	assert.True(f.Enabled)
}

// TestSqliteRepository_GetFeatureConcurrent shows that the sqlite database
// can be accessed concurrently
func TestSqliteRepository_GetFeatureConcurrent(t *testing.T) {
	assert := assert.New(t)

	repository, errChannels, _ := buildTestSqliteRepository(assert)
	defer repository.Close()

	wg := &sync.WaitGroup{}
	i := 1
	for {
		if i == 2000 {
			break
		}
		wg.Add(1)
		go testGetToggle(assert, i, repository, wg)
		i++
	}
	wg.Wait()

	select {
	case err := <-errChannels.errors:
		assert.NoError(err)
		return
	case err := <-errChannels.warnings:
		assert.NoError(err)
		return
	default:
	}
}

func testGetToggle(assert *assert.Assertions, i int, repository Repository, wg *sync.WaitGroup) {
	defer wg.Done()
	number := (i % 2) + 1
	key := fmt.Sprintf("dummy.feature%d", number)
	f := repository.GetToggle(key)
	assert.NotNil(f)

	if f != nil {
		assert.Equal(f.Name, key)
		assert.Equal(f.Enabled, number == 1)
	}
}
