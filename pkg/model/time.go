package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

type CustomTime time.Time

func (ct CustomTime) T() time.Time {
	return time.Time(ct)
}

const ctLayout = "2006-01-02 15:04:05 -0700 MST"

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	t := ct.T()
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", t.Format(ctLayout))), nil
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		*ct = CustomTime(time.Time{})
		return nil
	}
	t, err := time.Parse(`"`+ctLayout+`"`, s)
	if err != nil {
		return err
	}
	*ct = CustomTime(t)
	return nil
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
