package tmpl

type TemplateWatcher interface {
	// Watch watches a template file and sends a signal to the compiler
	// when a template needs to be recompiled.
	Watch(signal chan struct{})
}
