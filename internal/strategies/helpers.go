package strategies

import (
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/twmb/murmur3"
)

var VariantNormalizationSeed uint32 = 86028157

func round(f float64) int {
	if f < -0.5 {
		return int(f - 0.5)
	}
	if f > 0.5 {
		return int(f + 0.5)
	}
	return 0
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

func parameterAsFloat64(param interface{}) (result float64, ok bool) {
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

func NormalizedVariantValue(id string, groupId string, normalizer int, seed uint32) uint32 {
	hash := murmur3.SeedNew32(seed)
	hash.Write([]byte(groupId + ":" + id))
	hashCode := hash.Sum32()
	return hashCode%uint32(normalizer) + 1
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
	n := r.random.Intn(100) + 1
	r.Unlock()
	return n
}

func (r *rng) float() float64 {
	return float64(r.int())
}

func (r *rng) string() string {
	r.Lock()
	n := r.random.Intn(10000) + 1
	r.Unlock()
	return strconv.Itoa(n)
}

// newRng creates a new random number generator and uses a mutex
// internally to ensure safe concurrent reads.
func newRng() *rng {
	seed := time.Now().UnixNano() + int64(os.Getpid())
	return &rng{random: rand.New(rand.NewSource(seed))}
}
