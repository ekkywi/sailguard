package netenforce

import (
	"context"
	"errors"
)

var ErrUnsupported = errors.New("network enforce unsupported on this platform")

type Enforcer interface {
	Apply(ctx context.Context, spec Spec) error
	Clear(ctx context.Context) error
}