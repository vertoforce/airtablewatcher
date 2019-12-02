package airtabletasker

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

var ranFunction chan int

func performAction(tasker *Tasker, task *Task) {
	fmt.Println(task)
	select {
	case ranFunction <- 1:
	default:
	}
	// Make sure to change state after work is done!
	tasker.SetState(task.ID, "Done")
}

func TestNewTasker(t *testing.T) {
	tasker, err := NewTasker(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"), "Tasks")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	tasker.PollInterval = time.Second * 2

	// Register function
	ranFunction = make(chan int)
	tasker.RegisterFunction("ToDo", performAction)

	// Set a task to ToDo
	tasks, err := tasker.GetTasks()
	if len(tasks) == 0 || err != nil {
		t.Errorf("No tasks")
		return
	}
	err = tasker.SetState(tasks[0].ID, "ToDo")
	if err != nil {
		t.Errorf("Error setting state")
	}

	// Start tasker
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	go tasker.Start(ctx)

	// Wait for our function to run or context to be canceled
	select {
	case <-ranFunction:
	case <-ctx.Done():
		t.Errorf("Did not run function")
	}
	cancel()
}

func TestSetGetState(t *testing.T) {
	tasker, err := NewTasker(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"), "Tasks")
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	tasks, err := tasker.GetTasks()
	if len(tasks) == 0 || err != nil {
		t.Errorf("Failed to get tasks")
	}

	// Set and Get state of first task
	err = tasker.SetState(tasks[0].ID, "New")
	if state, _ := tasker.GetState(tasks[0].ID); state != "New" {
		t.Errorf("Incorrect state read")
	}
	tasker.SetState(tasks[0].ID, "Original")
	if state, _ := tasker.GetState(tasks[0].ID); state != "Original" {
		t.Errorf("Incorrect state read")
	}
}
