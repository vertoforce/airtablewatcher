package airtablewatcher

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

var ranFunction chan int

func performAction(ctx context.Context, watcher *Watcher, tableName string, row *Row) {
	fmt.Println(row)
	select {
	case ranFunction <- 1:
	default:
	}
	// Make sure to change state after work is done!
	watcher.SetRow("Tasks", row.ID, map[string]interface{}{"State": "Done"})
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
	watcher.RegisterFunction("Tasks", "State", []string{"ToDo"}, performAction)

	// Set a task to ToDo
	rows, err := watcher.GetRows("Tasks")
	if len(rows) == 0 || err != nil {
		t.Errorf("No tasks")
		return
	}
	err = watcher.SetRow("Tasks", rows[0].ID, map[string]interface{}{"State": "ToDo"})
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
	err = tasker.SetRow("Tasks", rows[0].ID, map[string]interface{}{"State": "New"})
	if row, _ := tasker.GetRow("Tasks", rows[0].ID); row.GetFieldString("State") != "New" {
		t.Errorf("Incorrect state read")
	}
	err = tasker.SetRow("Tasks", rows[0].ID, map[string]interface{}{"State": "Original"})
	if row, _ := tasker.GetRow("Tasks", rows[0].ID); row.GetFieldString("State") != "Original" {
		t.Errorf("Incorrect state read")
	}
}

func TestGetConfig(t *testing.T) {
	watcher, err := NewWatcher(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"))
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	val, err := watcher.GetConfig("TestKey")
	if err != nil {
		t.Errorf("Error: " + err.Error())
		return
	}
	if val != "TestValue" {
		t.Errorf("Didn't get correct value")
	}
}

var canceled = false

func CancelMe(ctx context.Context, watcher *Watcher, tableName string, row *Row) {
	watcher.SetRow(tableName, row.ID, map[string]interface{}{"State": "Processing"})
	// Don't return until canceled
	select {
	case <-ctx.Done():
		canceled = true
		watcher.SetRow(tableName, row.ID, map[string]interface{}{"State": "Error"})
		return
	}
}

func TestCancel(t *testing.T) {
	watcher, err := NewWatcher(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"))
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	watcher.PollInterval = time.Second * 2

	// Add function and start watcher
	watcher.RegisterFunction("Tasks", "State", []string{"ToDo"}, CancelMe, "Cancel")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watcher.Start(ctx)

	// Check if we have enough rows to test on
	rows, err := watcher.GetRows("Tasks")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if len(rows) == 0 {
		t.Errorf("Not enough rows to test on")
	}

	// Execute function
	watcher.SetRow("Tasks", rows[0].ID, map[string]interface{}{"State": "ToDo"})
	time.Sleep(time.Second * 3)

	// Cancel the function
	watcher.SetRow("Tasks", rows[0].ID, map[string]interface{}{"State": "Cancel"})

	// Check if the function was canceled
	time.Sleep(time.Second)
	if !canceled {
		t.Errorf("Did not cancel function")
	}

	// Execute function again
	canceled = false
	watcher.SetRow("Tasks", rows[0].ID, map[string]interface{}{"State": "ToDo"})
	time.Sleep(time.Second * 3)

	// Cancel entire watcher
	cancel()

	// Check if the function was canceled
	time.Sleep(time.Second)
	if !canceled {
		t.Errorf("Did not cancel function")
	}
}

// Test to make sure a job running already does not get triggered again
// For this test you must manually trigger a job and make sure it is not run again before it finishes
func TestConcurrentJob(t *testing.T) {
	watcher, err := NewWatcher(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"))
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	watcher.PollInterval = time.Second * 2

	performLongAction := func(ctx context.Context, watcher *Watcher, tableName string, row *Row) {
		fmt.Printf("Running long function %s\n", row.ID)
		time.Sleep(time.Second * 30)
		fmt.Println("Done")
	}

	// Register function
	watcher.RegisterFunction("Tasks", "State", []string{"ToDo"}, performLongAction)

	// Start tasker
	watcher.Start(context.Background())
}
