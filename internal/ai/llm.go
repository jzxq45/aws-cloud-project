package ai

import "context"

// LLMClient defines the interface for generating text from prompts.
type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
