package airtablewatcher

import (
	"fmt"
	"time"
)

// Defaults
const (
	AirtableDateFormat = "2006-01-02T15:04:05.000Z"
)

// More defaults
var (
	DefaultBlankTime = time.Date(1, 0, 0, 0, 0, 0, 0, time.UTC)
)

// Row Generic row from airtable
type Row struct {
	ID     string
	Fields interface{}
}

// GetField Get a generic field value from a row, returns nil if not found
func (r *Row) GetField(fieldName string) interface{} {
	// Attempt to cast and get state
	if res, ok := r.Fields.(map[string]interface{}); ok {
		if state, ok := res[fieldName]; ok {
			return state
		}
	}
	return nil
}

// GetFieldString Get string value from a row
func (r *Row) GetFieldString(fieldName string) string {
	// Attempt to cast and get state
	value := r.GetField(fieldName)
	if valueString, ok := value.(string); ok {
		return valueString
	}
	if valueBool, ok := value.(bool); ok {
		if valueBool {
			return "true"
		}
		return "false"
	}
	if valueFloat, ok := value.(float64); ok {
		return fmt.Sprintf("%f", valueFloat)
	}
	return ""
}

// GetFieldTime Get a field value in time format from a row, returns DefaultBlankTime if parsing fails
func (r *Row) GetFieldTime(fieldName string) time.Time {
	// Attempt to cast and get state
	timeStr := r.GetFieldString(fieldName)
	if timeStr == "" {
		return DefaultBlankTime
	}
	time, err := time.Parse(AirtableDateFormat, timeStr)
	if err == nil {
		return time
	}
	return DefaultBlankTime
}

// GetRows Get list of tasks in airtable
func (t *Watcher) GetRows(tableName string) ([]Row, error) {
	tasks := []Row{}
	err := t.AirtableClient.ListRecords(tableName, &tasks)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// GetRow Get airtable row
func (t *Watcher) GetRow(tableName, recordID string) (*Row, error) {
	row := &Row{}
	err := t.AirtableClient.RetrieveRecord(tableName, recordID, row)
	if err != nil {
		return nil, err
	}

	return row, nil
}

// SetRow Set provided fields for a row
func (t *Watcher) SetRow(tableName, recordID string, fields map[string]interface{}) error {
	return t.AirtableClient.UpdateRecord(tableName, recordID, fields, nil)
}
