package utils

import (
	set "github.com/deckarep/golang-set"
)

func ToSet(strings []string) set.Set {
	// First convert to array of []interface{} .. yikes!
	generics := make([]interface{}, len(strings))
	for i, v := range strings {
		generics[i] = v
	}

	return set.NewSetFromSlice(generics)
}

func ToStrings(generics []interface{}) []string {
	strings := make([]string, len(generics))
	for i, v := range generics {
		strings[i] = v.(string)
	}

	return strings
}
