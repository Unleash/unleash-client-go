package unleash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRepository_GetFeaturesFail tests that OnReady isn't fired unless
// /client/features has returned successfully.
func TestGetFetchURLPath(t *testing.T) {
	assert := assert.New(t)
	res := getFetchURLPath("")
	assert.Equal("./client/features", res)

	res = getFetchURLPath("myProject")
	assert.Equal("./client/features?project=myProject", res)
}
