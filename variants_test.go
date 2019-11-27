package unleash

import (
	"testing"

	"github.com/konfortes/unleash-client-go/v3/internal/api"
	"github.com/stretchr/testify/assert"
)

// TODO: MOCK after dev ends!
func TestVariants(t *testing.T) {
	assert := assert.New(t)

	client, err := NewClient(
		WithUrl("http://localhost:4242/api"),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
	)
	assert.Nil(err, "client should not return an error")

	go func() {
		for {
			select {
			case e := <-client.Errors():
				t.Fatalf("Unexpected error: %v", e)
			case w := <-client.Warnings():
				t.Fatalf("Unexpected warning: %v", w)
			case <-client.Count():
			case <-client.Sent():
			}
		}
	}()

	<-client.Registered()
	<-client.Ready()

	// feature exists
	enabled, value := client.GetVariant("color")
	assert.True(enabled)
	assert.Equal(value.Value, "blue")

	// feature exists with context
	// TODO: do

	// feature does not exist
	enabled, value = client.GetVariant("notexist")
	assert.False(enabled)
	assert.Nil(value)

	// feature does not exist with fallback
	enabled, value = client.GetVariant("notexist", WithVarFallback(api.VariantPayload{Type: "string", Value: "default"}))
	assert.True(enabled)
	assert.Equal(value.Value, "default")
}
