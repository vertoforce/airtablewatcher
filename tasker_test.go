package airtablewatcher

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

var ranFunction chan int

func performAction(watcher *Watcher, row *Row) {
	fmt.Println(row)
	select {
	case ranFunction <- 1:
	default:
	}
	// Make sure to change state after work is done!
	watcher.SetField("Tasks", row.ID, "State", "Done")
}

func TestNewWatcher(t *testing.T) {
	watcher, err := NewWatcher(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"))
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	watcher.PollInterval = time.Second * 2

	// Register function
	ranFunction = make(chan int)
	watcher.RegisterFunction("Tasks", "State", "ToDo", performAction)

	// Set a task to ToDo
	rows, err := watcher.GetRows("Tasks")
	if len(rows) == 0 || err != nil {
		t.Errorf("No tasks")
		return
	}
	err = watcher.SetField("Tasks", rows[0].ID, "State", "ToDo")
	if err != nil {
		t.Errorf("Error setting state")
	}

	// Start tasker
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	go watcher.Start(ctx)

	// Wait for our function to run or context to be canceled
	select {
	case <-ranFunction:
	case <-ctx.Done():
		t.Errorf("Did not run function")
	}
	cancel()
}

func TestSetGetState(t *testing.T) {
	tasker, err := NewWatcher(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"))
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	rows, err := tasker.GetRows("Tasks")
	if len(rows) == 0 || err != nil {
		t.Errorf("Failed to get tasks")
	}

	// Set and Get state of first task
	err = tasker.SetField("Tasks", rows[0].ID, "State", "New")
	if state, _ := tasker.GetField("Tasks", rows[0].ID, "State"); state != "New" {
		t.Errorf("Incorrect state read")
	}
	err = tasker.SetField("Tasks", rows[0].ID, "State", "Original")
	if state, _ := tasker.GetField("Tasks", rows[0].ID, "State"); state != "Original" {
		t.Errorf("Incorrect state read")
	}
}
