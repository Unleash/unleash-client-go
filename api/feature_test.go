package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureSegmentsResponse(t *testing.T) {
	assert := assert.New(t)
	featureResponse := FeatureResponse{
		Features: []Feature{},
		Segments: []Segment{
			{Id: 1, Constraints: []Constraint{
				{
					ContextName: "custom-id",
					Operator:    OperatorIn,
					Values:      []string{"custom-ctx"},
				}}},
			{Id: 2, Constraints: []Constraint{
				{
					ContextName: "age",
					Operator:    OperatorNumGte,
					Value:       "5",
				}}},
		}}

	segmentsMap := featureResponse.SegmentsMap()

	segmentOne := segmentsMap[1]
	segmentTwo := segmentsMap[2]
	segmentThree := segmentsMap[3]

	assert.Equal(segmentOne[0].Operator, OperatorIn)
	assert.Equal(segmentTwo[0].Operator, OperatorNumGte)
	assert.Nil(segmentThree)
}
