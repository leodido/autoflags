package autoflags

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	autoflagstesting "github.com/leodido/autoflags/testing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestOptions struct {
	Value string `flag:"value" flagdescr:"test value"`
}

func (o *TestOptions) Attach(c *cobra.Command) error { return nil }

func TestConcurrentCommandCreation(t *testing.T) {
	const numGoroutines = 100
	const numCommandsPerGoroutine = 10

	var wg sync.WaitGroup
	results := make(chan *scope, numGoroutines*numCommandsPerGoroutine)

	// Create commands concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numCommandsPerGoroutine; j++ {
				cmdName := fmt.Sprintf("test%d-%d", goroutineID, j)
				cmd := &cobra.Command{Use: cmdName}
				opts := &TestOptions{Value: cmdName}

				// This should be thread-safe
				Define(cmd, opts)

				// Get the scope and send it to results
				scope := getScope(cmd)
				results <- scope
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Collect all scopes
	scopes := make([]*scope, 0, numGoroutines*numCommandsPerGoroutine)
	for scope := range results {
		scopes = append(scopes, scope)
	}

	// Verify we got the expected number of scopes
	assert.Len(t, scopes, numGoroutines*numCommandsPerGoroutine)

	// Verify all scopes are unique (no shared state)
	scopeSet := make(map[*scope]bool)
	for _, scope := range scopes {
		assert.NotNil(t, scope)
		assert.False(t, scopeSet[scope], "Found duplicate scope - indicates shared state")
		scopeSet[scope] = true
	}

	// Verify each scope has its own viper instance
	viperSet := make(map[*viper.Viper]bool)
	for _, scope := range scopes {
		viper := scope.viper()
		assert.NotNil(t, viper)
		assert.False(t, viperSet[viper], "Found duplicate viper instance - indicates shared state")
		viperSet[viper] = true
	}
}

func TestCommandIsolation(t *testing.T) {
	// Test commands with same name get different scopes
	t.Run("same_name_different_scopes", func(t *testing.T) {
		cmd1 := &cobra.Command{Use: "test"}
		cmd2 := &cobra.Command{Use: "test"} // Same name!

		opts1 := &TestOptions{Value: "value1"}
		opts2 := &TestOptions{Value: "value2"}

		Define(cmd1, opts1)
		Define(cmd2, opts2)

		scope1 := getScope(cmd1)
		scope2 := getScope(cmd2)

		// Should have different scopes despite same command name
		assert.NotSame(t, scope1, scope2, "Commands with same name should have different scopes")

		// Should have different viper instances
		viper1 := scope1.viper()
		viper2 := scope2.viper()
		assert.NotSame(t, viper1, viper2, "Commands should have different viper instances")

		// Get initial state of second scope
		initialBoundEnvs2 := scope2.getBoundEnvs()

		// Modify one scope's boundEnvs
		scope1.setBound("test-flag")

		// Verify isolation
		updatedBoundEnvs1 := scope1.getBoundEnvs()
		updatedBoundEnvs2 := scope2.getBoundEnvs()

		assert.True(t, updatedBoundEnvs1["test-flag"], "First scope should have bound env")
		assert.False(t, updatedBoundEnvs2["test-flag"], "Second scope should not have bound env")
		assert.Equal(t, initialBoundEnvs2, updatedBoundEnvs2, "Second scope should be unchanged")
	})

	// Test parent-child command isolation
	t.Run("parent_child_isolation", func(t *testing.T) {
		parentCmd := &cobra.Command{Use: "parent"}
		childCmd := &cobra.Command{Use: "child"}

		// Simulate parent-child relationship (child inherits parent's context)
		parentCtx := parentCmd.Context()
		if parentCtx == nil {
			parentCtx = context.Background()
		}
		childCmd.SetContext(parentCtx)

		Define(parentCmd, &TestOptions{})
		Define(childCmd, &TestOptions{})

		parentScope := getScope(parentCmd)
		childScope := getScope(childCmd)

		// Even with context inheritance, should get different scopes
		assert.NotSame(t, parentScope, childScope, "Child command should have different scope than parent")
	})

	// Test concurrent access to same command
	t.Run("concurrent_same_command", func(t *testing.T) {
		cmd := &cobra.Command{Use: "concurrent"}
		Define(cmd, &TestOptions{})

		const numGoroutines = 50
		var wg sync.WaitGroup
		scopes := make([]*scope, numGoroutines)

		// Multiple goroutines getting scope from same command
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				scopes[index] = getScope(cmd)
			}(i)
		}

		wg.Wait()

		// All should get the same scope (since it's the same command)
		firstScope := scopes[0]
		require.NotNil(t, firstScope)

		for i, scope := range scopes {
			assert.Same(t, firstScope, scope, "All goroutines should get same scope for same command (index %d)", i)
		}
	})
}

func TestMemoryCleanup(t *testing.T) {
	if autoflagstesting.IsRaceOn() {
		t.Skip("Skipping memory test when race detector is on")
	}
	// This test verifies the pattern works and doesn't accumulate excessive memory

	// Force GC to get a clean baseline
	runtime.GC()
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	var initialMemStats, finalMemStats runtime.MemStats
	runtime.ReadMemStats(&initialMemStats)

	const numCommands = 1000

	// Create and discard many commands
	for i := 0; i < numCommands; i++ {
		cmd := &cobra.Command{Use: fmt.Sprintf("test%d", i)}
		opts := &TestOptions{Value: fmt.Sprintf("value%d", i)}

		Define(cmd, opts)
		scope := getScope(cmd)

		// Verify scope is created
		assert.NotNil(t, scope)
		assert.NotNil(t, scope.viper())

		// Verify scope isolation by adding some data
		scope.setBound(fmt.Sprintf("test-env-%d", i))
		boundEnvs := scope.getBoundEnvs()
		assert.True(t, boundEnvs[fmt.Sprintf("test-env-%d", i)])

		// Command and scope go out of scope here
	}

	// Force garbage collection multiple times
	for i := 0; i < 5; i++ {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}

	runtime.ReadMemStats(&finalMemStats)

	// Use TotalAlloc (cumulative) instead of Alloc (current) for more stable measurement
	memoryAllocatedMB := float64(finalMemStats.TotalAlloc-initialMemStats.TotalAlloc) / 1024 / 1024

	// Use HeapSys to check if heap grew significantly
	heapGrowthMB := float64(finalMemStats.HeapSys-initialMemStats.HeapSys) / 1024 / 1024

	t.Logf("Total memory allocated during test: %.2f MB", memoryAllocatedMB)
	t.Logf("Heap growth: %.2f MB", heapGrowthMB)
	t.Logf("Created %d commands", numCommands)

	// The main goal is to verify no excessive heap growth
	// Allow up to 50MB heap growth for 1000 commands (very generous)
	assert.Less(t, heapGrowthMB, 50.0,
		"Heap should not grow excessively (grew %.2f MB)", heapGrowthMB)

	// Verify memory was actually allocated (sanity check)
	assert.Greater(t, memoryAllocatedMB, 0.1,
		"Should have allocated some memory during test")

	// Additional check: verify current memory usage isn't excessive
	currentAllocMB := float64(finalMemStats.Alloc) / 1024 / 1024
	t.Logf("Current allocated memory: %.2f MB", currentAllocMB)

	// This is a very generous limit - in practice it should be much lower
	assert.Less(t, currentAllocMB, 200.0,
		"Current memory usage should be reasonable (%.2f MB)", currentAllocMB)
}
