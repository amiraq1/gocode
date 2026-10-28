package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"gocode/internal/provider"
)

type Tool struct {
	Name          string
	Description   string
	Params        json.RawMessage
	NeedsApproval bool
	Describe      func(args json.RawMessage) string // ملخص يُعرض عند طلب الموافقة
	Run           func(ctx context.Context, args json.RawMessage) (string, error)
}

type Approver func(summary string) bool

type Registry struct {
	tools   map[string]*Tool
	order   []string
	approve Approver
}

func NewRegistry(approve Approver) *Registry {
	return &Registry{tools: map[string]*Tool{}, approve: approve}
}

func (r *Registry) Register(t *Tool) {
	if _, dup := r.tools[t.Name]; !dup {
		r.order = append(r.order, t.Name)
	}
	r.tools[t.Name] = t
}

func (r *Registry) Defs() []provider.ToolDef {
	defs := make([]provider.ToolDef, 0, len(r.order))
	for _, n := range r.order {
		t := r.tools[n]
		defs = append(defs, provider.ToolDef{Type: "function", Function: provider.FunctionDef{
			Name: t.Name, Description: t.Description, Parameters: t.Params}})
	}
	return defs
}

// Execute لا يُرجع خطأ Go أبداً: الأخطاء تُعاد كنص كي يراها النموذج ويصحح نفسه.
func (r *Registry) Execute(ctx context.Context, name, args string) string {
	t, ok := r.tools[name]
	if !ok {
		return "error: unknown tool " + name
	}
	if t.NeedsApproval {
		summary := fmt.Sprintf("%s %s", name, args)
		if t.Describe != nil {
			summary = t.Describe(json.RawMessage(args))
		}
		if r.approve == nil || !r.approve(summary) {
			return "error: the user denied this action"
		}
	}
	out, err := t.Run(ctx, json.RawMessage(args))
	if err != nil {
		return "error: " + err.Error()
	}
	return TruncateOutput(out, MaxToolOutputBytes)
}
