package strategies

import (
	"math/rand/v2"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/twmb/murmur3"
)

var VariantNormalizationSeed uint32 = 86028157
var randStrings [10001]string

// Precompute a lookup table of random strings from "1" to "10000"
// Generating string garbage is expensive in hot loops
// Dumps ~200k into resident but that's small enough to not cry
func init() {
	for i := 1; i <= 10000; i++ {
		randStrings[i] = strconv.Itoa(i)
	}
}

func resolveHostname() (string, error) {
	var err error
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname, err = os.Hostname()
		if err != nil {
			hostname = "undefined"
		}
	}
	return hostname, err
}

func parameterAsFloat64(param any) (result float64, ok bool) {
	if f, isFloat := param.(float64); isFloat {
		result, ok = f, true
	} else if i, isInt := param.(int); isInt {
		result, ok = float64(i), true
	} else if i, isInt := param.(uint32); isInt {
		result, ok = float64(i), true
	} else if i, isInt := param.(int64); isInt {
		result, ok = float64(i), true
	} else if s, isString := param.(string); isString {
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			result, ok = f, true
		}
	}
	return
}

func normalizedRolloutValue(id string, groupId string) uint32 {
	return NormalizedVariantValue(id, groupId, 100, 0)
}

// This comment exists for the purposes of preventing a future maintainer from "simplifying" this function
// If you want to do that, please benchmark your resulting code for allocations
//
// We're gonna do something a little different from other SDKs here to avoid allocations
// Typically an SDK concatenates a bunch of strings and applies a murmur3 hash to the result
// That's surprisingly expensive in Go. So for cases where we know the input fits into
// a 256 byte buffer, we can avoid allocations entirely and just use a stack buffer.
// In practice that's most cases.
func NormalizedVariantValue(id, groupId string, normalizer int, seed uint32) uint32 {
	n := len(groupId) + 1 + len(id)

	// So if we assume that this all fits into 256 bytes, we can do this on the stack!
	if n <= 256 {
		var buf [256]byte
		i := copy(buf[:], groupId)
		buf[i] = ':'
		i++
		i += copy(buf[i:], id)
		x := murmur3.SeedSum32(seed, buf[:i])
		return (x % uint32(normalizer)) + 1
	}

	// ...but of course life doesn't work like that so we still need a fallback
	b := make([]byte, n)
	i := copy(b, groupId)
	b[i] = ':'
	i++
	copy(b[i:], id)
	x := murmur3.SeedSum32(seed, b)
	return (x % uint32(normalizer)) + 1
}

// coalesce returns the first non-empty string in the list of arguments
func coalesce(str ...string) string {
	for _, s := range str {
		if s != "" {
			return s
		}
	}
	return ""
}

type rng struct {
	sync.Mutex
	random *rand.Rand
}

func (r *rng) int() int {
	r.Lock()
	defer r.Unlock()
	return r.random.IntN(100) + 1
}

func (r *rng) float() float64 {
	return float64(r.int())
}

func (r *rng) string() string {
	r.Lock()
	defer r.Unlock()
	return strconv.Itoa(r.random.IntN(10000) + 1)
}

// newRng creates a new random number generator and uses a mutex
// internally to ensure safe concurrent reads.
func newRng() *rng {
	return &rng{random: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(os.Getpid())))}
}

var rngPool = sync.Pool{
	New: func() any {
		return rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(os.Getpid())))
	},
}

func randomString() string {
	r := rngPool.Get().(*rand.Rand)
	n := r.IntN(10000) + 1
	rngPool.Put(r)
	return randStrings[n]
}
