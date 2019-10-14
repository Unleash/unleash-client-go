package strategies

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/spaolacci/murmur3"
)

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

func normalizedValue(id string, groupId string) uint32 {
	value := fmt.Sprintf("%s:%s", groupId, id)
	hash := murmur3.New32()
	hash.Write([]byte(value))
	hashCode := hash.Sum32()
	return hashCode % uint32(100) + 1
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

// newRand creates a new random number generator
func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().Unix() + int64(os.Getpid())))
}
