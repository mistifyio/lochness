package cli

import (
	"encoding/json"
	"fmt"
)

// JMap is a generic resource
type JMap map[string]interface{}

// ID returns the id value
func (j JMap) ID() string {
	if id, ok := j["id"]; ok {
		return id.(string)
	}
	return ""
}

// String marshals into a json string
func (j JMap) String() string {
	buf, err := json.Marshal(&j)
	if err != nil {
		return ""
	}
	return string(buf)
}

// Print prints either the json string or just the id
func (j JMap) Print(json bool) {
	if json {
		fmt.Println(j)
	} else {
		fmt.Println(j.ID())
	}
}

// JMapSlice is an array of generic resources
type JMapSlice []JMap

// Len returns the length of the array
func (js JMapSlice) Len() int {
	return len(js)
}

// Less returns the comparsion of two elements
func (js JMapSlice) Less(i, j int) bool {
	return js[i].ID() < js[j].ID()
}

// Swap swaps two elements
func (js JMapSlice) Swap(i, j int) {
	js[j], js[i] = js[i], js[j]
}
