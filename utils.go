package unleash

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math/rand/v2"
	"os"
	"os/user"
	"sync"
)

func getTmpDirPath() string {
	return os.TempDir()
}

func generateInstanceId() string {
	prefix := ""

	if user, err := user.Current(); err == nil && user.Username != "" {
		prefix = user.Username
	} else {
		prefix = fmt.Sprintf("generated-%d-%d", rand.N(1000000), os.Getpid())
	}

	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return fmt.Sprintf("%s-%s", prefix, hostname)
	}
	return prefix
}

// https://github.com/google/uuid/blob/2d3c2a9cc518326daf99a383f07c4d3c44317e4d/version4.go#L47-L56
// https://github.com/hprose/hprose-go/blob/83de97da5004027694d321ca38c80fca3fac98c2/uuid.go#L91-L98
func getConnectionId() string {
	b := make([]byte, 16)
	cryptoRand.Read(b)

	b[6] = (b[6] & 0x0F) | 0x40
	b[8] = (b[8] & 0x3F) | 0x80

	uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)

	return uuid
}

func getFetchURLPath(projectName string) string {
	if projectName != "" {
		return fmt.Sprintf("./client/features?project=%s", projectName)
	}
	return "./client/features"
}

// WarnOnce is a type for handling warnings that should only be displayed once.
type WarnOnce struct {
	once sync.Once
}

// Warn logs the warning message once.
func (wo *WarnOnce) Warn(message string) {
	wo.once.Do(func() {
		fmt.Println("Warning:", message)
	})
}

// every returns true iff condition returns true for all elements in the input slice.
// This function will return false for empty slices (unlike the convention used in mathematical logic).
func every[T any](slice []T, condition func(T) bool) bool {
	if len(slice) == 0 {
		return false
	}
	for _, element := range slice {
		if !condition(element) {
			return false
		}
	}
	return true
}
