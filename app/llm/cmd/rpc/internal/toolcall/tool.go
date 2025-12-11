package toolcall

import (
	"context"
)


type Tool interface {
    Name() string
    Description() string
	ArgumentsJson() string
    Execute(ctx context.Context, argsJson string) (string, error)
}