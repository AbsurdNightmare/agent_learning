package agent

import (
	"context"
	"encoding/json"
	"os"

	"learning/model"
)

const checkpointPath = "checkpoint.json"

type CheckPoint struct {
	RunID           string          `json:"run_id"`
	StepID          int             `json:"step_id"`
	CurrentState    AgentState      `json:"current_state"`
	Messages        []model.Message `json:"messages"`
	PendingToolCall model.ToolCall  `json:"pending_tool_call"`
}

func LoadCheckPoint(ctx context.Context) (*CheckPoint, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	content, err := os.ReadFile(checkpointPath)
	if err != nil {
		return nil, err
	}

	var restored CheckPoint
	if err := json.Unmarshal(content, &restored); err != nil {
		return nil, err
	}
	return &restored, nil
}

func SaveCheckPoint(ctx context.Context, checkpoint *CheckPoint) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	content, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	return os.WriteFile(checkpointPath, content, 0644)
}
