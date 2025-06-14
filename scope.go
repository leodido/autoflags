package autoflags

import (
	"context"
	"sync"

	"maps"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cobra"
	spf13viper "github.com/spf13/viper"
)

// autoflagsContextKey is used to store scope in command context
type autoflagsContextKey struct{}

// scope holds per-command state for autoflags
type scope struct {
	v                 *spf13viper.Viper
	boundEnvs         map[string]bool
	customDecodeHooks map[string]mapstructure.DecodeHookFunc
	mu                sync.RWMutex
}

// getScope retrieves or creates a scope for the given command
func getScope(c *cobra.Command) *scope {
	ctx := c.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check if command already has scope
	if s, ok := ctx.Value(autoflagsContextKey{}).(*scope); ok {
		return s
	}

	// Create new scope (ensures isolation even with context inheritance)
	s := &scope{
		v:                 spf13viper.New(),
		boundEnvs:         make(map[string]bool),
		customDecodeHooks: make(map[string]mapstructure.DecodeHookFunc),
	}

	// Attach to command context
	newCtx := context.WithValue(ctx, autoflagsContextKey{}, s)
	c.SetContext(newCtx)

	return s
}

// viper returns the viper instance for the command
func (s *scope) viper() *spf13viper.Viper {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.v
}

// isEnvBound checks if an environment variable is already bound for this command
func (s *scope) isEnvBound(flagName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.boundEnvs[flagName]
}

// setBound marks an environment variable as bound for this command
func (s *scope) setBound(flagName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.boundEnvs[flagName] = true
}

// getBoundEnvs is for testing purposes only
func (s *scope) getBoundEnvs() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]bool, len(s.boundEnvs))
	maps.Copy(result, s.boundEnvs)

	return result
}

func (s *scope) setCustomDecodeHook(hookName string, hookFunc mapstructure.DecodeHookFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.customDecodeHooks[hookName] = hookFunc
}

func (s *scope) getCustomDecodeHook(hookName string) (hookFunc mapstructure.DecodeHookFunc, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hookFunc, ok = s.customDecodeHooks[hookName]

	return
}
