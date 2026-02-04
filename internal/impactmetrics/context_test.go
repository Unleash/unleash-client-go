package impactmetrics

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractEnvironmentFromValidHeaders(t *testing.T) {
	headers := http.Header{
		"Authorization": []string{"project:environment.hash"},
		"Content-Type":  []string{"application/json"},
	}

	env, ok := extractEnvironmentFromHeaders(headers)
	assert.True(t, ok)
	assert.Equal(t, "environment", env)
}

func TestExtractEnvironmentCaseInsensitiveHeaderKeys(t *testing.T) {
	headers := http.Header{}
	headers.Set("AUTHORIZATION", "project:environment.hash")
	headers.Set("Content-Type", "application/json")

	env, ok := extractEnvironmentFromHeaders(headers)
	assert.True(t, ok)
	assert.Equal(t, "environment", env)
}

func TestExtractEnvironmentHeaderNotPresent(t *testing.T) {
	headers := http.Header{}

	env, ok := extractEnvironmentFromHeaders(headers)
	assert.False(t, ok)
	assert.Equal(t, "", env)
}

func TestExtractEnvironmentPartIsEmpty(t *testing.T) {
	headers := http.Header{
		"Authorization": []string{"project:.hash"},
	}

	env, ok := extractEnvironmentFromHeaders(headers)
	assert.False(t, ok)
	assert.Equal(t, "", env)
}
