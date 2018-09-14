package unleash

import (
	"encoding/json"
	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/h2non/gock.v1"
	"os"
	"path/filepath"
	"testing"
	)

const mockHost = "http://unleash-apu"
const specFolder = "./testdata/client-specification/specifications"

var specIndex = filepath.Join(specFolder, "index.json")

type TestState struct {
	Version  int           `json:"version"`
	Features []api.Feature `json:"features"`
}

type TestCase struct {
	Description    string          `json:"description"`
	Context        context.Context `json:"context"`
	ToggleName     string          `json:"toggleName"`
	ExpectedResult bool            `json:"expectedResult"`
}

func (tc TestCase) RunWithClient(client *Client) func(*testing.T) {
	return func(t *testing.T) {
		client.WaitForReady()
		result := client.IsEnabled(tc.ToggleName, WithContext(tc.Context))
		assert.Equal(t, tc.ExpectedResult, result)
	}
}

type TestDefinition struct {
	Name  string    `json:"name"`
	State TestState `json:"state"`
	Tests []TestCase `json:"tests"`
}

func (td TestDefinition) Mock() (*Client, error) {
	gock.New(mockHost).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Response: api.Response{
				Version: td.State.Version,
			},
			Features: td.State.Features,
		})

	return NewClient(
		WithUrl(mockHost),
		WithAppName("clientSpecificationTest"),
		WithListener(MockedListener{}),
	)
}

func (td TestDefinition) Unmock() () {
	gock.OffAll()
}

func (td TestDefinition) Run(t *testing.T) {
	for _, test := range td.Tests {
		client, err := td.Mock()
		assert.NoError(t, err)
		t.Run(test.Description, test.RunWithClient(client))
		td.Unmock()
	}
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
	for _, def := range s.definitions {
		s.T().Run(def.Name, def.Run)
	}
}

func TestClientSpecificationSuite(t *testing.T) {
	suite.Run(t, new(ClientSpecificationSuite))
}
