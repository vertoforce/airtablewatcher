// Package airtabletasker makes it easy to run functions on airtable rows when a "State" is set
package airtabletasker

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
)

// Task Generic row from airtable
type Task struct {
	ID     string
	Fields interface{}
}

// Tasker configuration to watch airtable for a change in state
type Tasker struct {
	PollInterval  time.Duration
	AirtableTable string

	// Perform action functions in separate go routines
	Async bool

	airtableClient *airtable.Client
	airtableKey    string
	airtableBase   string
	timeout        time.Duration
	watchers       []watcher
}

// watching trigger and function
type watcher struct {
	trigger        string
	actionFunction ActionFunction
}

// ActionFunction Function that runs when triggered
type ActionFunction func(tasker *Tasker, airtableTask *Task)

// NewTasker Create new tasker to watch airtable
func NewTasker(airtableKey, airtableBase, tableName string) (*Tasker, error) {
	tasker := &Tasker{
		airtableKey:   airtableKey,
		airtableBase:  airtableBase,
		PollInterval:  DefaultAirtablePollInterval,
		AirtableTable: tableName,
		Async:         DefaultAsync,
	}
	err := tasker.connect()
	if err != nil {
		return nil, err
	}

	return tasker, nil
}

// connect or reconnect to airtable
func (t *Tasker) connect() error {
	airtableClient, err := airtable.New(t.airtableKey, t.airtableBase)
	if err != nil {
		return err
	}
	t.airtableClient = airtableClient

	return nil
}

// RegisterFunction Register a function to run on an airtable row when the state is changed to the trigger state.
func (t *Tasker) RegisterFunction(triggerState string, actionFunction ActionFunction) {
	t.watchers = append(t.watchers, watcher{triggerState, actionFunction})
}

// SetState Set state of airtable row
func (t *Tasker) SetState(id, state string) error {
	return t.airtableClient.UpdateRecord(t.AirtableTable, id, map[string]interface{}{
		"State": state,
	}, nil)
}

// GetState Get state of airtable row
func (t *Tasker) GetState(id string) (string, error) {
	var task Task
	err := t.airtableClient.RetrieveRecord(t.AirtableTable, id, &task)
	if err != nil {
		return "", err
	}

	// Attempt to pull state from task
	if state := getState(&task); state != "" {
		return state, nil
	}

	return "", errors.New("Could not parse state from task")
}

// GetTasks Get list of tasks in airtable
func (t *Tasker) GetTasks() ([]Task, error) {
	tasks := []Task{}
	err := t.airtableClient.ListRecords(t.AirtableTable, &tasks)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// Start watch airtable for triggers, blocking function.
func (t *Tasker) Start(ctx context.Context) {
	for {
		// Go through each task
		tasks, err := t.GetTasks()
		if err != nil {
			break
		}

		// Check each task
		for _, task := range tasks {
			// Attempt to pull state from task
			if state := getState(&task); state != "" {
				// Check each watcher
				for _, watcher := range t.watchers {
					if watcher.trigger == state {
						if t.Async {
							go watcher.actionFunction(t, &task)
						} else {
							watcher.actionFunction(t, &task)
						}
					}
				}
			}
		}

		// Check context
		select {
		case <-ctx.Done():
			break
		default:
		}

		time.Sleep(t.PollInterval)
	}

}

// getState pull state field from a task
func getState(task *Task) string {
	// Attempt to cast and get state
	if res, ok := task.Fields.(map[string]interface{}); ok {
		if state, ok := res["State"]; ok {
			if stateString, ok := state.(string); ok {
				return stateString
			}
		}
	}
	return ""
}
