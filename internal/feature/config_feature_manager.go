package feature

import (
	"context"
	"sync"
)

type ConfigFeatureManager struct {
	flags map[string]bool
	mu    sync.RWMutex
}

func (m *ConfigFeatureManager) IsEnabled(_ context.Context, feature string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.flags[feature]
}
