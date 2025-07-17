package gindump

import (
	"encoding/json"
	"strings"
)

var StringMaxLength = 0
var Newline = "" //"\n"
var Indent = 4

func FormatJsonBytes(data []byte, hiddenFields []string) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	v = removeHiddenFields(v, hiddenFields)

	return v, nil
}

// support
func FormatToJson(v interface{}, hiddenFields []string) (interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return FormatJsonBytes(data, hiddenFields)
}

func removeHiddenFields(v interface{}, hiddenFields []string) interface{} {
	if _, ok := v.(map[string]interface{}); !ok {
		return v
	}

	m := v.(map[string]interface{})

	// case insensitive key deletion
	for _, hiddenField := range hiddenFields {
		for k := range m {
			if strings.EqualFold(k, hiddenField) {
				delete(m, k)
			}
		}
	}

	return m
}
