package tools

import (
	"context"
	"sort"

	"github.com/openai/openai-go/v2"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() openai.FunctionParameters
	Call(ctx context.Context, rawArgs string) (string, error)
}

type Registry struct {
	byName map[string]Tool
}

func NewRegistry(ts ...Tool) *Registry {
	m := make(map[string]Tool, len(ts))
	for _, t := range ts {
		m[t.Name()] = t
	}
	return &Registry{byName: m}
}

func (r *Registry) Register(t Tool) {
	if r.byName == nil {
		r.byName = make(map[string]Tool)
	}
	r.byName[t.Name()] = t
}

func (r *Registry) Tools() []Tool {
	out := make([]Tool, 0, len(r.byName))
	for _, t := range r.byName {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.byName[name]
	return t, ok
}

func (r *Registry) AsOpenAITools() []openai.ChatCompletionToolUnionParam {
	ts := r.Tools()
	list := make([]openai.ChatCompletionToolUnionParam, 0, len(ts))
	for _, t := range ts {
		list = append(list, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name(),
			Description: openai.String(t.Description()),
			Parameters:  t.Parameters(),
		}))
	}
	return list
}

func (r *Registry) Execute(ctx context.Context, name string, rawArgs string) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", ErrUnknownTool(name)
	}
	return t.Call(ctx, rawArgs)
}

type unknownToolError struct{ name string }

func (e unknownToolError) Error() string { return "unknown tool: " + e.name }

func ErrUnknownTool(name string) error { return unknownToolError{name: name} }
