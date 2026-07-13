// Package lifecycle provides a process-wide "flush before hard exit" hook.
// os.Exit (called by logging.Fatal, and by any future hard-exit path) skips
// deferred functions, so anything that must run before such an exit —
// flushing telemetry, closing files, etc. — has to be registered here
// instead of relying on `defer` in main.
package lifecycle

import "sync"

// mu guards hook. logging.Fatal can run on more than one goroutine around
// the same time (e.g. a listener failure and a graceful shutdown both
// erroring), so RunShutdownHooks can be entered concurrently.
var mu sync.Mutex

// hook runs immediately before a hard process exit, once registered.
var hook func()

// RegisterShutdownHook registers the function to run before the process
// exits via RunShutdownHooks. There is only ever one caller that needs this
// (main's telemetry shutdown) — registering again replaces the previous hook.
func RegisterShutdownHook(h func()) {
	mu.Lock()
	defer mu.Unlock()
	hook = h
}

// RunShutdownHooks runs the registered hook, if any. Safe to call
// concurrently: the hook is snapshotted under mu before running, so two
// overlapping callers each run it once rather than racing.
func RunShutdownHooks() {
	mu.Lock()
	h := hook
	mu.Unlock()

	if h != nil {
		h()
	}
}
