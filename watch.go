package tmpl

type TemplateWatcher interface {
	// Spawn watches a template file and sends a signal to the compiler
	// when a template needs to be recompiled.
	Spawn(signal chan struct{})
}
