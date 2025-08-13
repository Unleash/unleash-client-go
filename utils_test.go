package unleash

import (
	"slices"
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
		allEven := every(numbers, func(item int) bool {
			return item%2 == 0
		})
		if !allEven {
			t.Errorf("Expected all numbers to be even, but got false")
		}
	})

	t.Run("All Long Strings", func(t *testing.T) {
		words := []string{"apple", "banana", "cherry"}
		allLong := every(words, func(item string) bool {
			return len(item) > 3
		})
		if !allLong {
			t.Errorf("Expected all words to be long, but got false")
		}
	})

	t.Run("Empty Slice", func(t *testing.T) {
		emptySlice := []int{}
		allEmpty := every(emptySlice, func(item int) bool {
			// This condition should not be reached for an empty slice.
			t.Errorf("Unexpected condition reached")
			return false
		})

		if allEmpty {
			t.Errorf("Expected an empty slice to return false, but got true")
		}
	})

	t.Run("Result should be false if one doesn't match the predicate", func(t *testing.T) {
		words := []string{"apple", "banana", "cherry", "he"}
		allLong := every(words, func(item string) bool {
			return len(item) > 3
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
		result := slices.Contains(arr, str)
		if !result {
			t.Errorf("Expected '%s' to be in the slice, but it was not found", str)
		}
	})

	t.Run("Element is not present in the slice", func(t *testing.T) {
		arr := []string{"apple", "banana", "cherry", "date", "fig"}
		str := "grape"
		result := slices.Contains(arr, str)
		if result {
			t.Errorf("Expected '%s' not to be in the slice, but it was found", str)
		}
	})

	t.Run("Empty slice should return false", func(t *testing.T) {
		arr := []string{}
		str := "apple"
		result := slices.Contains(arr, str)
		if result {
			t.Errorf("Expected an empty slice to return false, but it returned true")
		}
	})
}

func TestGetConnectionId(t *testing.T) {
	for range 100 {
		uuid := getConnectionId()

		t.Run("Correct length", func(t *testing.T) {
			if len(uuid) != 36 {
				t.Errorf("Expected UUID length to be 36, but got %d in %s", len(uuid), uuid)
			}
		})

		t.Run("UUIDv4 version", func(t *testing.T) {
			if uuid[14] != '4' {
				t.Errorf("Expected version 4, but got %c in %s", uuid[14], uuid)
			}
		})

		t.Run("UUIDv4 variant", func(t *testing.T) {
			variant := uuid[19]
			if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
				t.Errorf("Expected variant 10xx, but got %c in %s", variant, uuid)
			}
		})

		t.Run("Uniqueness", func(t *testing.T) {
			uuid2 := getConnectionId()
			if uuid == uuid2 {
				t.Errorf("Generated UUIDs are not unique: '%s' and '%s'", uuid, uuid2)
			}
		})
	}
}
