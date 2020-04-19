package airtablewatcher

import "errors"

// GetConfig Get value of config key
func (t *Watcher) GetConfig(key string) (string, error) {
	rows, err := t.GetRows(t.ConfigTableName)
	if err != nil {
		return "", err
	}

	for _, row := range rows {
		if thisKey := row.GetFieldString("Key"); thisKey == key {
			return row.GetFieldString("Value"), nil
		}
	}

	return "", errors.New("config key not found")
}
