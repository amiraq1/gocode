package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type Provider interface {
	Chat(ctx context.Context, msgs []Message, tools []ToolDef) (Message, error)
}

type OpenAI struct {
	BaseURL, APIKey, Model string
	HTTP                   *http.Client
}

func NewOpenAI(baseURL, key, model string) *OpenAI {
	return &OpenAI{BaseURL: baseURL, APIKey: key, Model: model,
		HTTP: &http.Client{Timeout: 5 * time.Minute}}
}

func (c *OpenAI) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (Message, error) {
	body := map[string]any{"model": c.Model, "messages": msgs}
	if len(tools) > 0 {
		body["tools"] = tools
	}
	b, err := json.Marshal(body)
	if err != nil {
		return Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return Message{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Message{}, fmt.Errorf("API %d: %s", resp.StatusCode, truncate(string(data), 500))
	}

	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return Message{}, fmt.Errorf("decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return Message{}, errors.New("empty response from model")
	}
	return out.Choices[0].Message, nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
