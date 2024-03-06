package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type CustomTime time.Time

func (ct CustomTime) T() time.Time {
	return time.Time(ct)
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	*ct = CustomTime(t)

	return nil
}

func (ct *CustomTime) MarshalJSON() ([]byte, error) {
	formattedTime := ct.T().Format("2006-01-02 15:04:05")
	return json.Marshal(formattedTime)
}

func (ct CustomTime) Value() (driver.Value, error) {
	return time.Time(ct), nil
}

func (ct *CustomTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*ct = CustomTime(v)
	default:
		return errors.New("type conversion to CustomTime failed")
	}

	return nil
}
