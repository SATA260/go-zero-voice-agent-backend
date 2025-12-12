package toolcall

import (
	"context"
)

type Tool interface {
	Name() string
	Description() string
	ArgumentsJson() string
	RequiresConfirmation() bool
	Scope() string
	Execute(ctx context.Context, argsJson string) (string, error)
}
