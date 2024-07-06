package tmpl

// FuncMapProvider is a struct type that returns its corresponding template functions.
// To be used in conjunction with the TemplateProvider interface.
type FuncMapProvider interface {
	TemplateFuncMap() FuncMap
}
