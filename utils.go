package unleash

import (
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"time"
)

func getTmpDirPath() string {
	return os.TempDir()
}

func generateInstanceId() string {
	prefix := ""

	if user, err := user.Current(); err == nil && user.Username != "" {
		prefix = user.Username
	} else {
		rand.Seed(time.Now().Unix())
		prefix = fmt.Sprintf("generated-%d-%d", rand.Intn(1000000), os.Getpid())
	}

	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return fmt.Sprintf("%s-%s", prefix, hostname)
	}
	return prefix
}

func getFetchURLPath(projectName string) string {
	if projectName != "" {
		return fmt.Sprintf("./client/features?project=%s", projectName)
	}
	return "./client/features"
}
