package domain

import "context"

type NLProcessor interface {
	Process(ctx context.Context, content string, metadata map[string]string) (string, error)
}
