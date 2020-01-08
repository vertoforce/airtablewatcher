// Package airtablewatcher makes it easy to watch an airtable for row changes and run a function on that row
package airtablewatcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fabioberger/airtable-go"
)

// Defaults
const (
	DefaultAirtablePollInterval = time.Second * 10
	DefaultAirtableTable        = "Tasks"
	DefaultConfigTableName      = "Config"

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

// Watcher configuration to watch airtable for a change in state
type Watcher struct {
	PollInterval time.Duration
	// Table for configuration items with Key,Value fields
	ConfigTableName string
	AirtableClient  *airtable.Client

	airtableKey  string
	airtableBase string
	timeout      time.Duration
	watchers     []watch
	ctx          context.Context
}

// watch trigger and function
type watch struct {
	tableName      string
	fieldName      string
	triggerValue   string
	cancelValues   []string
	actionFunction ActionFunction
}

// ActionFunction Function that runs when triggered
// If the row is changed off the trigger value while the function is still running, the context is canceled
type ActionFunction func(ctx context.Context, watcher *Watcher, tableName string, airtableRow *Row)

// NewWatcher Create new tasker to watch airtable
func NewWatcher(airtableKey, airtableBase string) (*Watcher, error) {
	watcher := &Watcher{
		airtableKey:     airtableKey,
		airtableBase:    airtableBase,
		PollInterval:    DefaultAirtablePollInterval,
		ConfigTableName: DefaultConfigTableName,
	}
	err := watcher.connect()
	if err != nil {
		return nil, err
	}

	return watcher, nil
}

// connect or reconnect to airtable
func (t *Watcher) connect() error {
	airtableClient, err := airtable.New(t.airtableKey, t.airtableBase)
	if err != nil {
		return err
	}
	t.AirtableClient = airtableClient

	return nil
}

// RegisterFunction Register a function to run on an airtable row when the state is changed to the trigger state.
// cancelValue will cancel the function when any of the cancelValues is matched
func (t *Watcher) RegisterFunction(tableName, fieldName, triggerValue string, actionFunction ActionFunction, cancelValue ...string) {
	t.watchers = append(t.watchers, watch{tableName, fieldName, triggerValue, cancelValue, actionFunction})
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

// Start watch airtable for triggers, blocking function.
// The context applies to all sub tasks, if the context is canceled, all registered functions will be cancelled
// TODO: Make threadsafe
func (t *Watcher) Start(ctx context.Context) error {
	t.ctx = ctx
	for {
		// Get all tables we need to scan
		tables := map[string]bool{}
		for _, watcher := range t.watchers {
			tables[watcher.tableName] = true
		}

		// Go through each row in each table
		for tableName := range tables {
			rows, err := t.GetRows(tableName)
			if err != nil {
				return err
			}

			// Check each row
			for _, row := range rows {
				// Check each watcher
				for _, watcher := range t.watchers {
					// Check tableName
					if watcher.tableName != tableName {
						continue
					}

					if row.GetFieldString(watcher.fieldName) == watcher.triggerValue { // We should run this action function!
						actionFunctionCtx, actionFunctionCancel := context.WithCancel(t.ctx)

						// Cancel context if fieldName =/= triggerValue
						go t.watchForCancel(actionFunctionCtx, &row, &watcher, actionFunctionCancel)

						// Call action
						watcher.actionFunction(actionFunctionCtx, t, tableName, &row)

						actionFunctionCancel()
					}
				}
			}
		}

		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		time.Sleep(t.PollInterval)
	}
}

// watchForCancel watches a row if it leaves the trigger value, if it does, cancels the context
func (t *Watcher) watchForCancel(ctx context.Context, row *Row, watcher *watch, actionFunctionCancel context.CancelFunc) {
	for {
		rowUpdated, err := t.GetRow(watcher.tableName, row.ID)
		if err != nil {
			return
		}
		value := rowUpdated.GetFieldString(watcher.fieldName)
		for _, cancelValue := range watcher.cancelValues {
			if value == cancelValue {
				// Cancel that action function
				actionFunctionCancel()
				return
			}
		}
		time.Sleep(t.PollInterval / 2) // Poll this at double the rate of full poll

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
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

// GetConfig Get value of config key
func (t *Watcher) GetConfig(key string) (string, error) {
	rows, err := t.GetRows(t.ConfigTableName)
	if err != nil {
		return "", err
	}

	for _, row := range rows {
		if thisKey := row.GetFieldString("Key"); thisKey == key {
			return row.GetFieldString("Value"), nil
		}
	}

	return "", errors.New("config key not found")
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
