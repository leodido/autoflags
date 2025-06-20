package internalscope

import (
	"context"
	"sync"

	"maps"

	"github.com/go-viper/mapstructure/v2"
	autoflagserrors "github.com/leodido/autoflags/errors"
	"github.com/spf13/cobra"
	spf13viper "github.com/spf13/viper"
)

// autoflagsContextKey is used to store scope in command context
type autoflagsContextKey struct{}

// Scope holds per-command state for autoflags
type Scope struct {
	v                 *spf13viper.Viper
	boundEnvs         map[string]bool
	customDecodeHooks map[string]mapstructure.DecodeHookFunc
	definedFlags      map[string]string
	mu                sync.RWMutex
}

// Get retrieves or creates a scope for the given command
func Get(c *cobra.Command) *Scope {
	ctx := c.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check if command already has scope
	if s, ok := ctx.Value(autoflagsContextKey{}).(*Scope); ok {
		return s
	}

	// Create new scope (ensures isolation even with context inheritance)
	s := &Scope{
		v:                 spf13viper.New(),
		boundEnvs:         make(map[string]bool),
		customDecodeHooks: make(map[string]mapstructure.DecodeHookFunc),
		definedFlags:      make(map[string]string),
	}

	// Attach to command context
	newCtx := context.WithValue(ctx, autoflagsContextKey{}, s)
	c.SetContext(newCtx)

	return s
}

// Viper returns the viper instance for the command
func (s *Scope) Viper() *spf13viper.Viper {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.v
}

// IsEnvBound checks if an environment variable is already bound for this command
func (s *Scope) IsEnvBound(flagName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.boundEnvs[flagName]
}

// SetBound marks an environment variable as bound for this command
func (s *Scope) SetBound(flagName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.boundEnvs[flagName] = true
}

// GetBoundEnvs is for testing purposes only
func (s *Scope) GetBoundEnvs() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]bool, len(s.boundEnvs))
	maps.Copy(result, s.boundEnvs)

	return result
}

func (s *Scope) SetCustomDecodeHook(hookName string, hookFunc mapstructure.DecodeHookFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.customDecodeHooks[hookName] = hookFunc
}

func (s *Scope) GetCustomDecodeHook(hookName string) (hookFunc mapstructure.DecodeHookFunc, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hookFunc, ok = s.customDecodeHooks[hookName]

	return
}

// AddDefinedFlag adds a flag to the set of defined flags for this scope, returning an error if it's a duplicate.
func (s *Scope) AddDefinedFlag(name, fieldPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existingPath, ok := s.definedFlags[name]; ok {
		return autoflagserrors.NewDuplicateFlagError(name, fieldPath, existingPath)
	}
	s.definedFlags[name] = fieldPath

	return nil
}
