package model

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomTime_Scan_Time(t *testing.T) {
	ct := &CustomTime{}
	now := time.Now()

	err := ct.Scan(now)
	require.NoError(t, err)
	assert.WithinDuration(t, now, ct.Time, time.Second)
}

func TestCustomTime_Scan_Nil(t *testing.T) {
	ct := &CustomTime{}

	err := ct.Scan(nil)
	require.NoError(t, err)
	assert.True(t, ct.Time.IsZero())
}

func TestCustomTime_Scan_String_RFC3339(t *testing.T) {
	ct := &CustomTime{}
	timeStr := "2024-01-15T10:30:00Z"

	err := ct.Scan(timeStr)
	require.NoError(t, err)
	assert.Equal(t, 2024, ct.Time.Year())
	assert.Equal(t, time.January, ct.Time.Month())
	assert.Equal(t, 15, ct.Time.Day())
}

func TestCustomTime_Scan_String_RFC3339Nano(t *testing.T) {
	ct := &CustomTime{}
	timeStr := "2024-01-15T10:30:00.123456789Z"

	err := ct.Scan(timeStr)
	require.NoError(t, err)
	assert.Equal(t, 2024, ct.Time.Year())
}

func TestCustomTime_Scan_String_SQLiteFormat(t *testing.T) {
	ct := &CustomTime{}
	timeStr := "2024-01-15 10:30:00"

	err := ct.Scan(timeStr)
	require.NoError(t, err)
	assert.Equal(t, 2024, ct.Time.Year())
	assert.Equal(t, time.January, ct.Time.Month())
	assert.Equal(t, 15, ct.Time.Day())
}

func TestCustomTime_Value(t *testing.T) {
	now := time.Now()
	ct := CustomTime{Time: now}

	value, err := ct.Value()
	require.NoError(t, err)
	assert.Equal(t, now, value)
}

func TestCustomTime_MarshalJSON(t *testing.T) {
	ct := CustomTime{Time: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)}

	data, err := json.Marshal(ct)
	require.NoError(t, err)
	assert.Contains(t, string(data), "2024-01-15")
}

func TestCustomTime_UnmarshalJSON(t *testing.T) {
	ct := &CustomTime{}
	jsonData := `"2024-01-15T10:30:00Z"`

	err := ct.UnmarshalJSON([]byte(jsonData))
	require.NoError(t, err)
	assert.Equal(t, 2024, ct.Time.Year())
	assert.Equal(t, time.January, ct.Time.Month())
	assert.Equal(t, 15, ct.Time.Day())
}

func TestCustomTime_UnmarshalJSON_Null(t *testing.T) {
	ct := &CustomTime{}
	jsonData := `null`

	err := ct.UnmarshalJSON([]byte(jsonData))
	// null 应该保持零值
	require.NoError(t, err)
}

func TestCustomTime_Now(t *testing.T) {
	ct := Now()
	assert.False(t, ct.Time.IsZero())
	assert.WithinDuration(t, time.Now(), ct.Time, time.Second)
}

func TestCustomTime_Scan_DriverValue(t *testing.T) {
	// 测试实现 driver.Valuer 接口
	var _ driver.Valuer = CustomTime{}
}

func TestCustomTime_Scan_Scanner(t *testing.T) {
	// 测试实现 sql.Scanner 接口
	var _ interface{ Scan(interface{}) error } = &CustomTime{}
}

func TestCustomTime_Scan_InvalidFormat(t *testing.T) {
	ct := &CustomTime{}
	// 无效格式应该返回错误
	err := ct.Scan("invalid-date-format")
	assert.Error(t, err)
}

func TestCustomTime_MarshalJSON_InStruct(t *testing.T) {
	type TestStruct struct {
		CreatedAt CustomTime `json:"createdAt"`
	}

	ts := TestStruct{
		CreatedAt: CustomTime{Time: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)},
	}

	data, err := json.Marshal(ts)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"createdAt"`)
	assert.Contains(t, string(data), "2024-06-15")
}

func TestCustomTime_UnmarshalJSON_InStruct(t *testing.T) {
	type TestStruct struct {
		CreatedAt CustomTime `json:"createdAt"`
	}

	jsonData := `{"createdAt":"2024-06-15T12:00:00Z"}`

	var ts TestStruct
	err := json.Unmarshal([]byte(jsonData), &ts)
	require.NoError(t, err)
	assert.Equal(t, 2024, ts.CreatedAt.Year())
	assert.Equal(t, time.June, ts.CreatedAt.Month())
	assert.Equal(t, 15, ts.CreatedAt.Day())
}

func TestCustomTime_Before(t *testing.T) {
	earlier := CustomTime{Time: time.Now().Add(-1 * time.Hour)}
	later := CustomTime{Time: time.Now()}

	assert.True(t, earlier.Time.Before(later.Time))
	assert.False(t, later.Time.Before(earlier.Time))
}

func TestCustomTime_After(t *testing.T) {
	earlier := CustomTime{Time: time.Now().Add(-1 * time.Hour)}
	later := CustomTime{Time: time.Now()}

	assert.True(t, later.Time.After(earlier.Time))
	assert.False(t, earlier.Time.After(later.Time))
}

func TestCustomTime_Equal(t *testing.T) {
	now := time.Now()
	ct1 := CustomTime{Time: now}
	ct2 := CustomTime{Time: now}

	assert.True(t, ct1.Time.Equal(ct2.Time))
}

func TestCustomTime_Scan_MultipleFormats(t *testing.T) {
	formats := []string{
		"2024-01-15 10:30:00",
		"2024-01-15T10:30:00Z",
		"2024-01-15T10:30:00+08:00",
		"2024-01-15T10:30:00.123456789Z",
		"2024-01-15 10:30:00.123456789",
	}

	for _, format := range formats {
		ct := &CustomTime{}
		err := ct.Scan(format)
		require.NoError(t, err, "Failed to parse: %s", format)
		assert.Equal(t, 2024, ct.Time.Year(), "Year mismatch for: %s", format)
	}
}

func TestCustomTime_ZeroValue(t *testing.T) {
	var ct CustomTime
	assert.True(t, ct.Time.IsZero())
}

func TestCustomTime_IsZero(t *testing.T) {
	ct := CustomTime{}
	assert.True(t, ct.Time.IsZero())

	ct = Now()
	assert.False(t, ct.Time.IsZero())
}
