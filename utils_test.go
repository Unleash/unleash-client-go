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
	t.Run("All Even Integers", func(t *testing.T) {
		numbers := []int{2, 4, 6, 8, 10}
		allEven := every(numbers, func(item interface{}) bool {
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
	})

	t.Run("All Long Strings", func(t *testing.T) {
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
	})

	t.Run("Empty Slice", func(t *testing.T) {
		emptySlice := []int{}
		allEmpty := every(emptySlice, func(item interface{}) bool {
			// This condition should not be reached for an empty slice.
			t.Errorf("Unexpected condition reached")
			return false
		})

		if allEmpty {
			t.Errorf("Expected an empty slice to return false, but got true")
		}
	})

	t.Run("invalid inout", func(t *testing.T) {
		invalidInput := "string"
		result := every(invalidInput, func(item interface{}) bool {
			// This condition should not be reached for an empty slice.
			return true
		})

		if result == true {
			t.Errorf("Expected result to be false")
		}
	})

	t.Run("Result should be false if one doesn't match the predicate", func(t *testing.T) {
		words := []string{"apple", "banana", "cherry", "he"}
		allLong := every(words, func(item interface{}) bool {
			str, ok := item.(string)
			if !ok {
				t.Errorf("Expected a string, got %T", item)
				return false
			}
			return len(str) > 3
		})
		if allLong == true {
			t.Errorf("Expected all words to be long, but got false")
		}
	})
}

func TestContains(t *testing.T) {
	t.Run("Element is present in the slice", func(t *testing.T) {
		arr := []string{"apple", "banana", "cherry", "date", "fig"}
		str := "banana"
		result := contains(arr, str)
		if !result {
			t.Errorf("Expected '%s' to be in the slice, but it was not found", str)
		}
	})

	t.Run("Element is not present in the slice", func(t *testing.T) {
		arr := []string{"apple", "banana", "cherry", "date", "fig"}
		str := "grape"
		result := contains(arr, str)
		if result {
			t.Errorf("Expected '%s' not to be in the slice, but it was found", str)
		}
	})

	t.Run("Empty slice should return false", func(t *testing.T) {
		arr := []string{}
		str := "apple"
		result := contains(arr, str)
		if result {
			t.Errorf("Expected an empty slice to return false, but it returned true")
		}
	})
}

