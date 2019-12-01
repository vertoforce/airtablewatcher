# Airtable Tasker

Airtable tasker is a simple library to make it easy to run backend go code based on an airtable frontend.

The library watches the airtable field named `State` and runs a function when the state is changed to the watching state.

## Usage

```go
func performAction(tasker *Tasker, airtableEntry interface{}) {
    fmt.Println(airtableEntry)
    // Make sure to change state after work is done!
    tasker.SetState("Processed")
}

tasker := NewTasker("airtable_key", "airtable_base")
tasker.RegisterFunction("Processing", performAction())
```
