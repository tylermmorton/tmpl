package tmpl

type Watcher interface {
	// WatchSignal watches a template file and sends a signal to the compiler
	// when a template needs to be recompiled.
	// If any errors occur, they are sent to the error channel.
	WatchSignal(signal chan struct{}, ch chan error)
}
