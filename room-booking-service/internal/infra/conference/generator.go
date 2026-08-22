package conference

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type generator struct {
	baseURL string
}

func NewGenerator(baseURL string) (*generator, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("conference link generator - new: base URL is required")
	}

	return &generator{baseURL: baseURL}, nil
}

func (generator *generator) Generate(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("conference link generator - generate: %w", err)
	}

	return generator.baseURL + "/" + uuid.NewString(), nil
}
