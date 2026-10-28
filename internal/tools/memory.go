package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"gocode/internal/session"
)

type rememberArgs struct {
	Note string `json:"note"`
}

func MemoryTools(store *session.Store) []*Tool {
	return []*Tool{
		{
			Name:        "remember",
			Description: "Save a concise, important project fact or user preference to persistent project memory so it is remembered in future sessions.",
			Params:      json.RawMessage(`{"type":"object","properties":{"note":{"type":"string","description":"Concise note to remember about this project"}},"required":["note"]}`),
			Run: func(_ context.Context, raw json.RawMessage) (string, error) {
				var a rememberArgs
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", fmt.Errorf("bad arguments: %w", err)
				}
				if err := store.AppendMemory(a.Note); err != nil {
					return "", err
				}
				return "saved to project memory: " + a.Note, nil
			},
		},
	}
}
