package cli

import (
	"encoding/json"
	"fmt"
)

type JMap map[string]interface{}

func (j JMap) ID() string {
	return j["id"].(string)
}

func (j JMap) String() string {
	buf, err := json.Marshal(&j)
	if err != nil {
		return ""
	}
	return string(buf)
}

func (j JMap) Print(json bool) {
	if json {
		fmt.Println(j)
	} else {
		fmt.Println(j.ID())
	}
}

type JMapSlice []JMap

func (js JMapSlice) Len() int {
	return len(js)
}

func (js JMapSlice) Less(i, j int) bool {
	return js[i].ID() < js[j].ID()
}

func (js JMapSlice) Swap(i, j int) {
	js[j], js[i] = js[i], js[j]
}
