package types

import (
	"encoding/json"
	"errors"
)

// JSON是一个包装JSON . rawmessage的自定义类型。
// 用于在数据库中存储JSON数据。
type JSON json.RawMessage

// TODO: Value implements the driver.Valuer interface.

// Scan 实现了 sql.Scanner 接口。
func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	result := json.RawMessage{}
	err := json.Unmarshal(bytes, &result)
	*j = JSON(result)
	return err
}

// MarshalJSON 实现了 json.Marshaler 接口。
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 实现了 json.Unmarshaler 接口。
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSON: UnmarshalJSON on nil pointer")
	}
	*j = JSON(data)
	return nil
}

// ToString 将 JSON 转换为字符串。
func (j JSON) ToString() string {
	if len(j) == 0 {
		return "{}"
	}
	return string(j)
}

// Map 将 JSON 转换为映射。
func (j JSON) Map() (map[string]interface{}, error) {
	if len(j) == 0 {
		return map[string]interface{}{}, nil
	}

	var m map[string]interface{}
	err := json.Unmarshal(j, &m)
	return m, err
}
