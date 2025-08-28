//go:build norace
// +build norace

package unleash

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/Unleash/unleash-go-sdk/v5/context"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const mockHost = "http://unleash-apu"
const specFolder = "./testdata/client-specification/specifications"

var specIndex = filepath.Join(specFolder, "index.json")
var specNotImplemented = []string{""}

type TestState struct {
	Version  int           `json:"version"`
	Features []api.Feature `json:"features"`
	Segments []api.Segment `json:"segments"`
}

type TestCase struct {
	Description    string          `json:"description"`
	Context        context.Context `json:"context"`
	ToggleName     string          `json:"toggleName"`
	ExpectedResult bool            `json:"expectedResult"`
}

type expectedVariantResult struct {
	api.Variant
	// SpecFeatureEnabled represents the spec's feature_enabled field which has a
	// different JSON field name than api.Variant
	SpecFeatureEnabled bool `json:"feature_enabled"`
}

type VariantTestCase struct {
	Description    string                 `json:"description"`
	Context        context.Context        `json:"context"`
	ToggleName     string                 `json:"toggleName"`
	ExpectedResult *expectedVariantResult `json:"expectedResult"`
}

type Runner interface {
	GetDescription() string
	RunWithClient(*Client) func(*testing.T)
}

func (tc TestCase) GetDescription() string {
	return tc.Description
}

func (tc TestCase) RunWithClient(client *Client) func(*testing.T) {
	return func(t *testing.T) {
		client.WaitForReady()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			// Call IsEnabled concurrently with itself to catch
			// potential data races with go test -race.
			client.IsEnabled(tc.ToggleName, WithContext(tc.Context))
			wg.Done()
		}()
		result := client.IsEnabled(tc.ToggleName, WithContext(tc.Context))
		wg.Wait()
		assert.Equal(t, tc.ExpectedResult, result)
	}
}

func (vtc VariantTestCase) GetDescription() string {
	return vtc.Description
}

func (vtc VariantTestCase) RunWithClient(client *Client) func(*testing.T) {
	client.staticContext = &vtc.Context
	return func(t *testing.T) {
		client.WaitForReady()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			// Call IsEnabled concurrently with itself to catch
			// potential data races with go test -race.
			client.GetVariant(vtc.ToggleName, WithVariantContext(vtc.Context))
			wg.Done()
		}()
		result := client.GetVariant(vtc.ToggleName, WithVariantContext(vtc.Context))
		wg.Wait()
		assert.Equal(t, vtc.ExpectedResult.Enabled, result.Enabled)
		// copy over the FeatureEnabled field with different JSON tag
		vtc.ExpectedResult.FeatureEnabled = vtc.ExpectedResult.SpecFeatureEnabled
		assert.Equal(t, &vtc.ExpectedResult.Variant, result)
	}
}

type TestDefinition struct {
	Name         string            `json:"name"`
	State        TestState         `json:"state"`
	Tests        []TestCase        `json:"tests"`
	VariantTests []VariantTestCase `json:"variantTests"`
}

func (td TestDefinition) Mock(listener interface{}) (*Client, error) {
	gock.New(mockHost).
		Post("/client/register").
		Reply(200)

	// Check if this is a delta format test by checking if the spec file contains events
	specFile := filepath.Join(specFolder, td.Name+".json")
	data, err := os.ReadFile(specFile)
	if err != nil {
		return nil, err
	}
	
	// Parse to detect format type
	var rawSpec map[string]interface{}
	if err := json.Unmarshal(data, &rawSpec); err != nil {
		return nil, err
	}
	
	// Check if this is a delta format by looking for events in state
	if state, ok := rawSpec["state"].(map[string]interface{}); ok {
		if events, ok := state["events"].([]interface{}); ok && len(events) > 0 {
			// This is a delta format test - mock delta API response
			gock.New(mockHost).
				Get("/client/features").
				Reply(200).
				JSON(map[string]interface{}{
					"events": events,
				})
		} else {
			// Regular format test - use traditional FeatureResponse
			gock.New(mockHost).
				Get("/client/features").
				Reply(200).
				JSON(api.FeatureResponse{
					Response: api.Response{
						Version: td.State.Version,
					},
					Features: td.State.Features,
					Segments: td.State.Segments,
				})
		}
	} else {
		// Fallback to regular format if state structure is unexpected
		gock.New(mockHost).
			Get("/client/features").
			Reply(200).
			JSON(api.FeatureResponse{
				Response: api.Response{
					Version: td.State.Version,
				},
				Features: td.State.Features,
				Segments: td.State.Segments,
			})
	}

	return NewClient(
		WithUrl(mockHost),
		WithAppName("clientSpecificationTest"),
		WithListener(listener),
	)
}

func (td TestDefinition) Unmock() {
	gock.OffAll()
}

func (td TestDefinition) Run(t *testing.T) {
	runTest := func(test Runner) {
		listener := &MockedListener{}
		listener.On("OnReady").Return()
		listener.On("OnRegistered", mock.AnythingOfType("ClientData")).Return()
		listener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return()
		listener.On("OnError", mock.AnythingOfType("*errors.errorString")).Return()

		client, err := td.Mock(listener)
		assert.NoError(t, err)
		t.Run(test.GetDescription(), test.RunWithClient(client))
		client.Close()

		listener.AssertCalled(t, "OnReady")
		listener.AssertCalled(t, "OnRegistered", mock.AnythingOfType("ClientData"))

		td.Unmock()
	}

	for _, test := range td.Tests {
		runTest(test)
	}

	for _, test := range td.VariantTests {
		runTest(test)
	}
}

func (td TestDefinition) IsImplemented() bool {
	for _, name := range specNotImplemented {
		if name == td.Name {
			return false
		}
	}

	return true
}

type ClientSpecificationSuite struct {
	suite.Suite
	definitions []TestDefinition
}

func (s ClientSpecificationSuite) loadTestDefinition(testFile string) TestDefinition {
	test, err := os.Open(filepath.Join(specFolder, testFile))
	s.NoError(err)
	defer test.Close()
	var testDef TestDefinition
	dec := json.NewDecoder(test)
	err = dec.Decode(&testDef)
	s.NoError(err)
	return testDef
}

func (s *ClientSpecificationSuite) SetupTest() {
	index, err := os.Open(specIndex)
	s.NoError(err)
	defer index.Close()

	var testFiles []string
	dec := json.NewDecoder(index)
	err = dec.Decode(&testFiles)
	s.NoError(err)

	for _, testFile := range testFiles {
		s.definitions = append(s.definitions, s.loadTestDefinition(testFile))
	}
}

func (s ClientSpecificationSuite) TestClientSpecification() {
	for _, td := range s.definitions {
		if td.IsImplemented() {
			s.T().Run(td.Name, td.Run)
		}
	}
}

func TestClientSpecificationSuite(t *testing.T) {
	suite.Run(t, new(ClientSpecificationSuite))
}
