package agent

import (
	"context"
	"encoding/json"
	"os"

	"learning/model"
)

const checkpointPath = "checkpoint.json"

func LoadCheckPoint(ctx context.Context) (*model.CheckPoint, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	content, err := os.ReadFile(checkpointPath)
	if err != nil {
		return nil, err
	}

	var restored model.CheckPoint
	if err := json.Unmarshal(content, &restored); err != nil {
		return nil, err
	}
	return &restored, nil
}

func SaveCheckPoint(ctx context.Context, checkpoint *model.CheckPoint) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	content, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	return os.WriteFile(checkpointPath, content, 0644)
}
