package airtablewatcher

import (
	"context"
	"fmt"
	"os"
	"time"
)

func printTask(ctx context.Context, watcher *Watcher, tableName string, row *Row) {
	fmt.Printf("Running code on %v", row)
	// Make sure to change state after work is done!
	watcher.SetRow("Tasks", row.ID, map[string]interface{}{"State": "Done"})
}

func Example() {
	tasker, err := NewWatcher(os.Getenv("AIRTABLE_KEY"), os.Getenv("AIRTABLE_BASE"))
	if err != nil {
		return
	}
	tasker.PollInterval = time.Second * 5

	// Register function
	tasker.RegisterFunction("Tasks", "State", []string{"ToDo"}, printTask)

	// Start tasker
	tasker.Start(context.Background())
}
