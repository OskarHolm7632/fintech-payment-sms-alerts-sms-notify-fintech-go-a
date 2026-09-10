package paymentalerts

import (
	"encoding/json"
	"os"
	"sync"
)

type JSONLAuditor struct {
	Path string
	mu   sync.Mutex
}

func (a *JSONLAuditor) Append(record AuditRecord) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	file, err := os.OpenFile(a.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(record)
}
