package unleash

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_defaultsToNoopListener(t *testing.T) {
	result := Initialize(
		WithAppName("my-application"),
		WithUrl("http://localhost:4242"),
		WithCustomHeaders(http.Header{"Authorization": {"*:development.code"}}),
	)

	if result != nil {
		t.Fail()
	}
	res := IsEnabled("test", WithFallback(false))
	assert.Equal(t, false, res)

	assert.IsType(t, &NoopListener{}, defaultClient.errorListener)
}
