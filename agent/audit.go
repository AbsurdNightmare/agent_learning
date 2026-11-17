package agent

import (
	"encoding/json"
	"os"
	"time"
)

const auditPath = "audit.jsonl"

type AuditLog struct {
	Time     string `json:"time"`
	RunID    string `json:"run_id"`
	Action   string `json:"action"`
	ToolName string `json:"tool_name"`
	Args     string `json:"args,omitempty"`
}

func WriteAudit(log AuditLog) error {
	log.Time = time.Now().Format(time.RFC3339)

	data, err := json.Marshal(log)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	return nil
}
