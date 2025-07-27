package config

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is an alias of time.Duration used for deserializing time string from json 
type Duration time.Duration

func (duration Duration) UnmarshalJSON(b []byte) error {
	var unmarshalledJson interface{}

	err := json.Unmarshal(b, &unmarshalledJson)
	if err != nil {
		return err
	}

	switch value := unmarshalledJson.(type) {
	case float64:
		duration = Duration(time.Duration(value))
	case string:
		d, err := time.ParseDuration(value)
		duration = Duration(d)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid duration: %#v", unmarshalledJson)
	}

	return nil
}
