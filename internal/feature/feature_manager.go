package feature

import "context"

type FeatureManager interface {
	IsEnabled(ctx context.Context, feature string) bool
}
