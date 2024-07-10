package unleash

import (
	"fmt"
	"os"
	"reflect"
	"sync"
)

func getTmpDirPath() string {
	return os.TempDir()
}

func getFetchURLPath(projectName string) string {
	if projectName != "" {
		return fmt.Sprintf("./client/features?project=%s", projectName)
	}
	return "./client/features"
}

func contains(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}

// WarnOnce is a type for handling warnings that should only be displayed once.
type WarnOnce struct {
	once sync.Once
}

// Warn logs the warning message once.
func (wo *WarnOnce) Warn(message string) {
	wo.once.Do(func() {
		fmt.Println("Warning:", message)
	})
}

func every(slice interface{}, condition func(interface{}) bool) bool {
	sliceValue := reflect.ValueOf(slice)

	if sliceValue.Kind() != reflect.Slice {
		fmt.Println("Input is not a slice returning false")
		return false
	}

	if sliceValue.Len() == 0 {
		return false
	}

	for i := 0; i < sliceValue.Len(); i++ {
		element := sliceValue.Index(i).Interface()
		if !condition(element) {
			return false
		}
	}
	return true
}
