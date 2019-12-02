package airtabletasker

import (
	"context"
	"fmt"
	"os"
	"time"
)

func printTask(tasker *Tasker, task Task) {
	fmt.Printf("Running code on %v", task)
	// Make sure to change state after work is done!
	tasker.SetState(task.ID, "Done")
}

func Example() {
	tasker, err := NewTasker(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"))
	if err != nil {
		return
	}
	tasker.PollInterval = time.Second * 5

	// Register function
	tasker.RegisterFunction("ToDo", printTask)

	// Start tasker
	tasker.Start(context.Background())
}
