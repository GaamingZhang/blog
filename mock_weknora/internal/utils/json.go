package utils

import (
	"encoding/json"
	"fmt"

	jsonschema "github.com/google/jsonschema-go/jsonschema"
)

// ToJSON 将任意类型转换为 JSON 字符串
func ToJSON(v interface{}) string {
	json, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(json)
}

func GenerateSchema[T any]() json.RawMessage {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("failed to generate schema: %v", err))
	}

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal schema: %v", err))
	}

	return schemaBytes
}
