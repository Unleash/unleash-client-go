package impactmetrics

import (
	"net/http"
	"strings"
)

type StaticContext struct {
	AppName     string
	Environment string
}

// Authorization header format: project:environment.hash
func extractEnvironmentFromHeaders(headers http.Header) (string, bool) {
	auth := headers.Get("Authorization")
	if auth == "" {
		return "", false
	}

	parts := strings.SplitN(auth, ":", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", false
	}

	envParts := strings.SplitN(parts[1], ".", 2)
	if envParts[0] == "" {
		return "", false
	}

	return envParts[0], true
}

func BuildImpactMetricContext(customHeaders http.Header, appName, environment string) StaticContext {
	ctx := StaticContext{
		AppName:     appName,
		Environment: environment,
	}

	if customHeaders != nil {
		if env, ok := extractEnvironmentFromHeaders(customHeaders); ok {
			ctx.Environment = env
		}
	}

	return ctx
}
