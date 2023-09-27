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
func TestEvery(t *testing.T) {
	// Test case 1: Check if all integers in the slice are even.
	numbers1 := []int{2, 4, 6, 8, 10}
	allEven := every(numbers1, func(item interface{}) bool {
		num, ok := item.(int)
		if !ok {
			t.Errorf("Expected an integer, got %T", item)
			return false
		}
		return num%2 == 0
	})
	if !allEven {
		t.Errorf("Expected all numbers to be even, but got false")
	}

	// Test case 2: Check if all strings in the slice have more than 3 characters.
	words := []string{"apple", "banana", "cherry"}
	allLong := every(words, func(item interface{}) bool {
		str, ok := item.(string)
		if !ok {
			t.Errorf("Expected a string, got %T", item)
			return false
		}
		return len(str) > 3
	})
	if !allLong {
		t.Errorf("Expected all words to be long, but got false")
	}

	// Test case 3: Check an empty slice.
	emptySlice := []int{}
	allEmpty := every(emptySlice, func(item interface{}) bool {
		// This condition should not be reached for an empty slice.
		t.Errorf("Unexpected condition reached")
		return false
	})

	if allEmpty {
		t.Errorf("Expected an empty slice to return false, but got true")
	}
}

