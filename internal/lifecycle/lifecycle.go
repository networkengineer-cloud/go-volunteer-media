// Package lifecycle provides a process-wide "flush before hard exit" hook
// registry. os.Exit (called by logging.Fatal, and by any future hard-exit
// path) skips deferred functions, so anything that must run before such an
// exit — flushing telemetry, closing files, etc. — has to be registered
// here instead of relying on `defer` in main.
package lifecycle

import "sync"

// mu guards hooks. logging.Fatal can run on more than one goroutine around
// the same time (e.g. a listener failure and a graceful shutdown both
// erroring), so RunShutdownHooks can be entered concurrently.
var mu sync.Mutex

// hooks run synchronously, in registration order, immediately before a hard
// process exit.
var hooks []func()

// RegisterShutdownHook registers a function to run before the process exits
// via RunShutdownHooks.
func RegisterShutdownHook(hook func()) {
	mu.Lock()
	defer mu.Unlock()
	hooks = append(hooks, hook)
}

// RunShutdownHooks runs every registered hook, in registration order. Safe
// to call concurrently: hooks is snapshotted under mu before running, so
// two overlapping callers each run the hooks once rather than racing on the
// underlying slice.
func RunShutdownHooks() {
	mu.Lock()
	toRun := append([]func(){}, hooks...)
	mu.Unlock()

	for _, hook := range toRun {
		hook()
	}
}
