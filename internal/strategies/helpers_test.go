package strategies

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveHostname(t *testing.T) {
	envHostname := os.Getenv("HOSTNAME")
	osHostname, err := os.Hostname()

	assert.Nil(t, err, "os.Hostname should not return an error")

	hostname, err := resolveHostname()

	assert.Nil(t, err, "resolveHostname should not return an error")

	if envHostname != "" {
		assert.Equal(t, envHostname, hostname)
	} else {
		assert.Equal(t, osHostname, hostname)
	}
}

func TestParameterAsFloat64(t *testing.T) {
	goodData := map[interface{}]float64{
		"30":    30.0,
		"-0.01": -0.01,
		42:      42.0,
		0.1:     0.1,
	}

	for input, expected := range goodData {
		actual, ok := parameterAsFloat64(input)
		if !ok {
			t.Errorf("Not OK: %v", input)
		}
		assert.InDelta(t, actual, expected, 0.0000001)
	}

	badData := map[interface{}]float64{
		"pizza": -1.0,
		"0.0.1": -1.0,
	}

	for input, _ := range badData {
		_, ok := parameterAsFloat64(input)
		assert.False(t, ok)
	}
}

func TestNormalizedValue(t *testing.T) {
	assert.Equal(t, uint32(73), normalizedValue("123", "gr1"))
	assert.Equal(t, uint32(25), normalizedValue("999", "groupX"))
}

func TestCoalesce(t *testing.T) {
	assert.Equal(t, "", coalesce())
	assert.Equal(t, "foo", coalesce("foo"))
	assert.Equal(t, "bar", coalesce("", "bar"))
}

func TestNewRng(t *testing.T) {
	rng := newRng()

	wg := sync.WaitGroup{}

	testGen := func(n int) {
		for i := 0; i < n; i++ {
			randomInt := rng.int()
			assert.True(t, randomInt >= 0 && randomInt <= 100)

			randomString := rng.string()
			assert.True(t, len(randomString) <= 5)

			randomFloat := rng.float()
			assert.True(t, randomFloat > 0.0 && randomFloat <= 100.0)
		}
		wg.Done()
	}

	goRoutines := 20
	wg.Add(goRoutines)
	for j := 0; j < goRoutines; j++ {
		go testGen(100)
	}
	wg.Wait()
}

func BenchmarkNormalizedValue(b *testing.B) {
	// Add two sub-benchmarks since the compiler allows
	// strings and byte slices up to 32 bytes to be allocated
	// on the stack when they do not escape.

	const (
		smallId    = "1234567"
		smallGroup = "group42"
	)

	var (
		largeId    = strings.Repeat("a", 16)
		largeGroup = strings.Repeat("b", 16)
	)

	b.Run("value less than 32 bytes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = normalizedValue(smallId, smallGroup)
		}
	})

	b.Run("value greater than 32 bytes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = normalizedValue(largeId, largeGroup)
		}
	})
}
