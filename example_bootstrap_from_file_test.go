package unleash_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v4"
	"github.com/Unleash/unleash-client-go/v4/context"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func Test_bootstrapFromFile(t *testing.T) {
	a := assert.New(t)
	demoReader, err := os.Open("demo_app_toggles.json")
	if err != nil {
		t.Fail()
	}
	gock.New("http://foo.com").
		Post("/client/register").
		Persist().
		Reply(200)
		// Read the file into a byte slice
	featuresReader, err := os.Open("demo_app_toggles.json")
	if err != nil {
		t.Fail()
	}
	byteValue, _ := ioutil.ReadAll(featuresReader)
	// Convert the byte slice to a string
	jsonStr := string(byteValue)

	// Use the string as the body of the Gock request
	gock.New("http://foo.com").
		Get("/client/features").Persist().Reply(200).BodyString(jsonStr)
	err = unleash.Initialize(
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithRefreshInterval(5*time.Second),
		unleash.WithDisableMetrics(true),
		unleash.WithStorage(&unleash.BootstrapStorage{Reader: demoReader}),
		unleash.WithUrl("http://foo.com"),
	)

	if err != nil {
		t.Fail()
	}

	enabled := unleash.IsEnabled("DateExample", unleash.WithContext(context.Context{}))
	fmt.Printf("feature is enabled? %v\n", enabled)
	a.True(enabled)
	err = unleash.Close()
	a.Nil(err)
}
