package utils

import "fmt"

func CompareMapString(oldMap, newMap map[string]string) bool {
	var old string
	var new string
	for k, v := range oldMap {
		old += fmt.Sprintf("%s=%s", k, v)
	}
	for k, v := range newMap {
		new += fmt.Sprintf("%s=%s", k, v)
	}
	return old == new
}
