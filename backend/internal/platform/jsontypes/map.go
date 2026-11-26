package jsontypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Map map[string]any

func (m Map) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func (m *Map) Scan(value any) error {
	switch typed := value.(type) {
	case nil:
		*m = Map{}
	case []byte:
		if len(typed) == 0 {
			*m = Map{}
			return nil
		}
		return json.Unmarshal(typed, m)
	case string:
		if typed == "" {
			*m = Map{}
			return nil
		}
		return json.Unmarshal([]byte(typed), m)
	case map[string]any:
		*m = Map(typed)
	default:
		return fmt.Errorf("unsupported jsontypes.Map source %T", value)
	}
	return nil
}

func (m Map) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]any(m))
}

func (m *Map) UnmarshalJSON(data []byte) error {
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*m = Map(value)
	return nil
}
