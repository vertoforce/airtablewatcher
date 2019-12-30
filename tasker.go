// Package airtablewatcher makes it easy to watch an airtable for row changes and run a function on that row
package airtablewatcher

import (
	"context"
	"errors"
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
		for tableName, _ := range tables {
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
					if GetFieldFromRow(&row, watcher.fieldName) == watcher.triggerValue {
						// We should run this action function!

						actionFunctionCtx, actionFunctionCancel := context.WithCancel(t.ctx)
						watchForCancelCtx, watchForCancelCancel := context.WithCancel(actionFunctionCtx)

						// Cancel context if fieldName =/= triggerValue
						go t.watchForCancel(watchForCancelCtx, &row, &watcher, actionFunctionCancel)

						// Call action
						watcher.actionFunction(actionFunctionCtx, t, tableName, &row)

						actionFunctionCancel()
						watchForCancelCancel()
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
		value, _ := t.GetField(watcher.tableName, row.ID, watcher.fieldName)
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

// GetState Get state of airtable row
func (t *Watcher) GetField(tableName, recordID, fieldName string) (string, error) {
	var row Row
	err := t.AirtableClient.RetrieveRecord(tableName, recordID, &row)
	if err != nil {
		return "", err
	}

	// Attempt to pull state from task
	if state := GetFieldFromRow(&row, fieldName); state != "" {
		return state, nil
	}

	return "", errors.New("Could not parse state from task")
}

// SetField Attempt to set string field for a task
func (t *Watcher) SetField(tableName, recordID, fieldName, fieldVal string) error {
	return t.AirtableClient.UpdateRecord(tableName, recordID, map[string]interface{}{
		fieldName: fieldVal,
	}, nil)
}

// GetConfig Get value of config key
func (t *Watcher) GetConfig(key string) (string, error) {
	rows, err := t.GetRows(t.ConfigTableName)
	if err != nil {
		return "", err
	}

	for _, row := range rows {
		if thisKey := GetFieldFromRow(&row, "Key"); thisKey == key {
			return GetFieldFromRow(&row, "Value"), nil
		}
	}

	return "", errors.New("config key not found")
}

// GetFieldFromRow Attempt to get string field from task
func GetFieldFromRow(row *Row, fieldName string) string {
	// Attempt to cast and get state
	if res, ok := row.Fields.(map[string]interface{}); ok {
		if state, ok := res[fieldName]; ok {
			if stateString, ok := state.(string); ok {
				return stateString
			}
			if stateBool, ok := state.(bool); ok {
				if stateBool {
					return "true"
				}
				return "false"
			}
		}
	}
	return ""
}

func GetFieldFromRowTime(row *Row, fieldName string) time.Time {
	// Attempt to cast and get state
	if res, ok := row.Fields.(map[string]interface{}); ok {
		if state, ok := res[fieldName]; ok {
			if timeStr, ok := state.(string); ok {
				time, err := time.Parse(AirtableDateFormat, timeStr)
				if err == nil {
					return time
				}
			}
		}
	}
	return time.Date(1, 0, 0, 0, 0, 0, 0, time.UTC)
}
