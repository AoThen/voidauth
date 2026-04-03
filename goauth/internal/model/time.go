package model

import (
	"database/sql/driver"
	"time"
)

// CustomTime 自定义时间类型，支持 SQLite 时间格式
type CustomTime struct {
	time.Time
}

// Scan 实现 sql.Scanner 接口
func (ct *CustomTime) Scan(value interface{}) error {
	if value == nil {
		ct.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		ct.Time = v
		return nil
	case string:
		// 尝试多种时间格式解析
		formats := []string{
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999+07:00",
			"2006-01-02 15:04:05.999999999Z",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02T15:04:05.999999999Z",
			"2006-01-02T15:04:05.999999999-07:00",
			"2006-01-02T15:04:05.999999999+07:00",
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
		}

		var err error
		for _, format := range formats {
			ct.Time, err = time.Parse(format, v)
			if err == nil {
				return nil
			}
		}
		return err
	}

	return nil
}

// Value 实现 driver.Valuer 接口
func (ct CustomTime) Value() (driver.Value, error) {
	return ct.Time, nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return ct.Time.MarshalJSON()
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	return ct.Time.UnmarshalJSON(data)
}

// Now 返回当前时间
func Now() CustomTime {
	return CustomTime{Time: time.Now()}
}
