package strategies

import (
	"crypto/md5"
	"fmt"
	"io"
	"strconv"
)

func round(f float64) int {
	if f < -0.5 {
		return int(f - 0.5)
	}
	if f > 0.5 {
		return int(f + 0.5)
	}
	return 0
}

func parameterAsFloat64(param interface{}) (result float64, ok bool) {
	if f, isFloat := param.(float64); isFloat {
		result, ok = f, true
	} else if i, isInt := param.(int); isInt {
		result, ok = float64(i), true
	} else if i, isInt := param.(int64); isInt {
		result, ok = float64(i), true
	} else if s, isString := param.(string); isString {
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			result, ok = f, true
		}
	}
	return
}

func normalizedValue(id string, groupId string) int64 {
	value := fmt.Sprintf("%s:%s", groupId, id)
	hash := md5.New()
	io.WriteString(hash, value)
	hex := fmt.Sprintf("%x", hash.Sum(nil))
	hashCode, _ := strconv.ParseInt(hex[len(hex)-4:], 16, 32)
	return hashCode % 100
}
