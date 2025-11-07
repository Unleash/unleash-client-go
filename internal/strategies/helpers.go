package strategies

import (
	"math/rand/v2"
	"os"
	"strconv"
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
	random *rand.Rand
}

func (r *rng) int() int {
	return r.random.IntN(100) + 1
}

func (r *rng) float() float64 {
	return float64(r.int())
}

func (r *rng) string() string {
	return strconv.Itoa(r.random.IntN(10000) + 1)
}

// newRng creates a new random number generator that is safe for concurrent use.
// Uses math/rand/v2 which provides lock-free concurrent access via atomic operations.
func newRng() *rng {
	seed := uint64(time.Now().UnixNano()) + uint64(os.Getpid())
	return &rng{random: rand.New(rand.NewPCG(seed, 0))}
}
