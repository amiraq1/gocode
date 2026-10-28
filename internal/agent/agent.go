package agent

import (
	"context"
	"fmt"

	"gocode/internal/provider"
	"gocode/internal/tools"
)

type Agent struct {
	llm      provider.Provider
	reg      *tools.Registry
	system   string
	msgs     []provider.Message
	MaxSteps int
	OnTool   func(name, args string)
}

func New(llm provider.Provider, reg *tools.Registry, system string, maxSteps int) *Agent {
	a := &Agent{llm: llm, reg: reg, system: system, MaxSteps: maxSteps}
	a.Reset()
	return a
}

func (a *Agent) Messages() []provider.Message {
	out := make([]provider.Message, len(a.msgs))
	copy(out, a.msgs)
	return out
}

func (a *Agent) SetMessages(msgs []provider.Message) {
	if len(msgs) == 0 {
		a.Reset()
		return
	}
	a.msgs = make([]provider.Message, len(msgs))
	copy(a.msgs, msgs)
	if a.msgs[0].Role == "system" {
		a.msgs[0].Content = a.system
	} else {
		a.msgs = append([]provider.Message{{Role: "system", Content: a.system}}, a.msgs...)
	}
}

func (a *Agent) Send(ctx context.Context, input string) (string, error) {
	saved := len(a.msgs)
	a.msgs = append(a.msgs, provider.Message{Role: "user", Content: input})
	for step := 0; step < a.MaxSteps; step++ {
		reply, err := a.llm.Chat(ctx, a.msgs, a.reg.Defs())
		if err != nil {
			a.msgs = a.msgs[:saved] // التراجع عن الدور غير المكتمل لحماية الجلسة
			return "", err
		}
		a.msgs = append(a.msgs, reply)
		if len(reply.ToolCalls) == 0 {
			return reply.Content, nil
		}
		for _, tc := range reply.ToolCalls {
			if a.OnTool != nil {
				a.OnTool(tc.Function.Name, tc.Function.Arguments)
			}
			result := a.reg.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			a.msgs = append(a.msgs, provider.Message{Role: "tool", ToolCallID: tc.ID, Content: result})
		}
	}
	a.msgs = a.msgs[:saved]
	return "", fmt.Errorf("stopped after %d steps without a final answer", a.MaxSteps)
}
