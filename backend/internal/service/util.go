package service

import (
	"encoding/json"
	"strings"
)

func jsonUnmarshal(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}

func SplitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
