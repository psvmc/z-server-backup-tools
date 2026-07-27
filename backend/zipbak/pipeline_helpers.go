package zipbak

import (
	"encoding/json"
	"os"
)

func parseStatusJSON(out string) (Status, error) {
	var st Status
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		return Status{}, err
	}
	return st, nil
}

func osMkdirAll(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
