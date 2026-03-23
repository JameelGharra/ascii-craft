package utils

import (
	"encoding/json"
	"fmt"
)

func stringify(item any) string {
	if item == nil {
		return "nil"
	}

	switch t := item.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", item)
	default:
		d, err := json.Marshal(t)
		if err != nil {
			return string(d)
		}
	}
	return fmt.Sprintf("%s", item)
}
func formatAssertMsg(msg string, args ...any) {
	fmt.Printf("%s: ", msg)
	for _, item := range args {
		fmt.Printf("%v ", stringify(item))
	}
	fmt.Println()
}

func Assert(truth bool, message string, args ...any) {
	if !truth {
		formatAssertMsg(message, args...)
		panic("assertion failed: " + message)
	}
}
