package unleash

import (
	"fmt"
	"math/rand"
	"net/url"
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

func getFetchURLPath(projectName, environment string) string {
	u, _ := url.Parse("./client/features")

	q := u.Query()

	if projectName != "" {
		q.Set("project", projectName)
	}

	if environment != "" {
		q.Set("environment", environment)
	}

	u.RawQuery = q.Encode()

	return u.String()
}
