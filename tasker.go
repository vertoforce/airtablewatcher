// Package airtablewatcher makes it easy to watch an airtable for row changes and run a function on that row
package airtablewatcher

import (
	"context"
	"time"

	"github.com/fabioberger/airtable-go"
)

// Defaults
const (
	DefaultAirtablePollInterval = time.Second * 10
	DefaultAirtableTable        = "Tasks"
	DefaultConfigTableName      = "Config"
)

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

// watch is a an event we are watching for including a specific trigger and action function
type watch struct {
	tableName      string
	fieldName      string
	triggerValues  []string
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
func (t *Watcher) RegisterFunction(tableName, fieldName string, triggerValues []string, actionFunction ActionFunction, cancelValue ...string) {
	t.watchers = append(t.watchers, watch{tableName, fieldName, triggerValues, cancelValue, actionFunction})
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

					for _, triggerValue := range watcher.triggerValues {
						if row.GetFieldString(watcher.fieldName) == triggerValue { // We should run this action function!
							actionFunctionCtx, actionFunctionCancel := context.WithCancel(t.ctx)

							// Cancel context if fieldName =/= triggerValue
							go t.watchForCancel(actionFunctionCtx, &row, &watcher, actionFunctionCancel)

							// Call action
							watcher.actionFunction(actionFunctionCtx, t, tableName, &row)

							actionFunctionCancel()
							break
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

// watchForCancel watches a row if it changes to a cancel value, if it does, cancels the context
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
