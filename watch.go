package tmpl

type TemplateWatcher interface {
	// Watch watches a templateProvider file and sends a signal to the CompilerOptions
	// when a templateProvider needs to be recompiled.
	Watch(callback func() error)
}
