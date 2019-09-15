package unleash

import (
	"testing"

	"github.com/Unleash/unleash-client-go/v3/internal/api"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestClientWithoutListener(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{})

	client, err := NewClient(
		WithUrl(mockerServer),
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
	client.Close()
	assert.True(gock.IsDone(), "there should be no more mocks")
}
