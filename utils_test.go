package unleash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetFetchURLPath verifies that getFetchURLPath returns the correct path
func TestGetFetchURLPath(t *testing.T) {
	assert := assert.New(t)
	res := getFetchURLPath("")
	assert.Equal("./client/features", res)

	res = getFetchURLPath("myProject")
	assert.Equal("./client/features?project=myProject", res)
}
