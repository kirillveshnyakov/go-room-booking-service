package conference

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type linkGenerator struct {
	baseURL string
}

func NewLinkGenerator(baseURL string) (*linkGenerator, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("conference link generator - new: base URL is required")
	}

	return &linkGenerator{baseURL: baseURL}, nil
}

func (generator *linkGenerator) Generate(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("conference link generator - generate: %w", err)
	}

	return generator.baseURL + "/" + uuid.NewString(), nil
}
