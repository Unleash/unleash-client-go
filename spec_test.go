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
	Version  int               `json:"version"`
	Features []api.Feature     `json:"features"`
	Segments []api.Segment     `json:"segments"`
	Events   []json.RawMessage `json:"events"`
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
	// Process events if present (for delta API tests)
	features := td.State.Features
	segments := td.State.Segments

	if len(td.State.Events) > 0 {
		// Process delta events to build features and segments
		features, segments = td.processDeltaEvents()
	}

	gock.New(mockHost).
		Post("/client/register").
		Reply(200)
	gock.New(mockHost).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Response: api.Response{
				Version: td.State.Version,
			},
			Features: features,
			Segments: segments,
		})

	return NewClient(
		WithUrl(mockHost),
		WithAppName("clientSpecificationTest"),
		WithListener(listener),
	)
}

func (td TestDefinition) processDeltaEvents() ([]api.Feature, []api.Segment) {
	features := make(map[string]api.Feature)
	segments := make(map[int]api.Segment)

	// Process each event
	for _, eventRaw := range td.State.Events {
		var eventType struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(eventRaw, &eventType); err != nil {
			continue
		}

		switch eventType.Type {
		case "hydration":
			var hydration struct {
				Features []api.Feature `json:"features"`
				Segments []api.Segment `json:"segments"`
			}
			if err := json.Unmarshal(eventRaw, &hydration); err == nil {
				// Reset state for hydration
				features = make(map[string]api.Feature)
				segments = make(map[int]api.Segment)

				for _, f := range hydration.Features {
					features[f.Name] = f
				}
				for _, s := range hydration.Segments {
					segments[s.Id] = s
				}
			}

		case "feature-updated":
			var update struct {
				Feature api.Feature `json:"feature"`
			}
			if err := json.Unmarshal(eventRaw, &update); err == nil {
				features[update.Feature.Name] = update.Feature
			}

		case "feature-removed":
			var removal struct {
				FeatureName string `json:"featureName"`
			}
			if err := json.Unmarshal(eventRaw, &removal); err == nil {
				delete(features, removal.FeatureName)
			}

		case "segment-updated":
			var update struct {
				Segment api.Segment `json:"segment"`
			}
			if err := json.Unmarshal(eventRaw, &update); err == nil {
				segments[update.Segment.Id] = update.Segment
			}

		case "segment-removed":
			var removal struct {
				SegmentId int `json:"segmentId"`
			}
			if err := json.Unmarshal(eventRaw, &removal); err == nil {
				delete(segments, removal.SegmentId)
			}
		}
	}

	// Convert maps to slices
	var featureList []api.Feature
	for _, f := range features {
		featureList = append(featureList, f)
	}

	var segmentList []api.Segment
	for _, s := range segments {
		segmentList = append(segmentList, s)
	}

	return featureList, segmentList
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
		listener.On("OnError", mock.Anything).Return()

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
