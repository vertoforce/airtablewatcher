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
	DefaultAsync                = false
	DefaultConfigTableName      = "Config"
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

	// Perform action functions in separate go routines
	Async bool

	airtableClient *airtable.Client
	airtableKey    string
	airtableBase   string
	timeout        time.Duration
	watchers       []watch
}

// watch trigger and function
type watch struct {
	tableName      string
	fieldName      string
	triggerValue   string
	actionFunction ActionFunction
}

// ActionFunction Function that runs when triggered
type ActionFunction func(watcher *Watcher, airtableRow *Row)

// NewWatcher Create new tasker to watch airtable
func NewWatcher(airtableKey, airtableBase string) (*Watcher, error) {
	watcher := &Watcher{
		airtableKey:     airtableKey,
		airtableBase:    airtableBase,
		PollInterval:    DefaultAirtablePollInterval,
		Async:           DefaultAsync,
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
	t.airtableClient = airtableClient

	return nil
}

// RegisterFunction Register a function to run on an airtable row when the state is changed to the trigger state.
func (t *Watcher) RegisterFunction(tableName, fieldName, triggerValue string, actionFunction ActionFunction) {
	t.watchers = append(t.watchers, watch{tableName, fieldName, triggerValue, actionFunction})
}

// GetRows Get list of tasks in airtable
func (t *Watcher) GetRows(tableName string) ([]Row, error) {
	tasks := []Row{}
	err := t.airtableClient.ListRecords(tableName, &tasks)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// Start watch airtable for triggers, blocking function.
// TODO: Make threadsafe
func (t *Watcher) Start(ctx context.Context) error {
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
					// Check fieldName and triggerValue
					if GetFieldFromRow(&row, watcher.fieldName) == watcher.triggerValue {
						if t.Async {
							go watcher.actionFunction(t, &row)
						} else {
							watcher.actionFunction(t, &row)
						}
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

// GetState Get state of airtable row
func (t *Watcher) GetField(tableName, recordID, fieldName string) (string, error) {
	var row Row
	err := t.airtableClient.RetrieveRecord(tableName, recordID, &row)
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
	return t.airtableClient.UpdateRecord(tableName, recordID, map[string]interface{}{
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
