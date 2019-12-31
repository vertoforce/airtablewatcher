# Airtable Watcher

Airtable watcher is a simple library to make it easy to run backend go code based on an airtable frontend.

The library watches an airtable base and runs a function when a field is changed to the `triggerValue`.

## Example

![example](example.gif)

## Usage

Set up an airtable base like [this one](https://airtable.com/shrrp5hz1D5JTb1HI).
Then write code to listen for any rows that have the `State` field set to `ToDo`.

```go
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
    tasker.RegisterFunction("Tasks", "State", "ToDo", printTask)

    // Start tasker
    tasker.Start(context.Background())
}
```
