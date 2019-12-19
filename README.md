# Airtable Watcher

Airtable watcher is a simple library to make it easy to run backend go code based on an airtable frontend.

The library watches the airtable field named `State` and runs a function when the state is changed to the watching state.

## Example

![example](example.gif)

## Usage

Set up an airtable base like [this one](https://airtable.com/shrrp5hz1D5JTb1HI).
Then write code to listen for any rows that have the State `ToDo`.

```go
func printTask(watcher *Watcher, row *Row) {
    fmt.Printf("Running code on %v", row)
    // Make sure to change state after work is done!
    watcher.SetField("Tasks", row.ID, "State", "Done")
}

func main() {
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
