# `tmpl`

> ⚠️ `tmpl` is currently working towards its first release

tmpl is a wrapper around Go's `html/template` package that aims to solve some of the pain points developers commonly run into while working with templates. This project attempts to improve the overall template workflow and offers a few helpful utilities for developers building html based applications:

- Two-way type safety when referencing templates in Go code and visa versa
- Nested templates and template fragments
- Template extensibility through compiler plugins
- Static analysis utilities such as template parse tree traversal
- Convenient but optional CLI for binding templates to Go code

*Roadmap & Idea List*

- Parsing and static analysis of the html in a template
- Automatic generation of [GoLand `{{ gotype: }}` annotations](https://www.jetbrains.com/help/go/integration-with-go-templates.html) when using the `tmpl` CLI
- Documentation on how to use `tmpl.Analyze` for parse tree traversal and static analysis of templates

##  🧰 Installation
```bash
go get github.com/tylermmorton/tmpl
```

To install the `tmpl` cli and scaffolding utilities:
```bash
go install github.com/tylermmorton/tmpl/cmd/tmpl
```

## 🌊 The Workflow

The `tmpl` workflow starts with a standard `html/template`. For more information on the syntax, see this [useful syntax primer from HashiCorp](https://developer.hashicorp.com/nomad/tutorials/templates/go-template-syntax).

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }} | torque</title>
</head>
<body>
    <form action="/login" method="post">
        <label for="username">Username</label>
        <input type="text" name="username" id="username" value="{{ .Username }}">

        <label for="password">Password</label>
        <input type="password" name="password" id="password" value="{{ .Password }}">

        <button type="submit">Login</button>
    </form>
</body>
```

### Dot Context

To start tying your template to your Go code, declare a struct that represents the "dot context" of the template. The dot context is the value of the "dot" (`{{ . }}`) in Go's templating language.

In this struct, any _exported_ fields (or methods attached via pointer receiver) will be accessible in your template from the all powerful dot.

```go
type LoginPage struct {
    Title    string // {{ .Title }}
    Username string // {{ .Username }}
    Password string // {{ .Password }}
}
```

### `TemplateProvider`

To turn your dot context struct into a target for the tmpl compiler, your struct type must implement the `TemplateProvider` interface:
```go
type TemplateProvider interface {
    TemplateText() string
}
```

The most straightforward approach is to embed the template into your Go program using the `embed` package from the standard library.

```go
import (
    _ "embed"
)

var (
    //go:embed login.tmpl.html
    tmplLoginPage string
)

type LoginPage struct { 
    ... 
}

func (*LoginPage) TemplateText() string {
    return tmplLoginPage
}
```

If you've opted into using the `tmpl` CLI, you can use the `//tmpl:bind` annotation on your dot context struct instead.

```go
//tmpl:bind login.tmpl.html
type LoginPage struct {
    ...
}
```

and run the utility:
```shell
tmpl bind . --outfile=tmpl.gen.go
```

> Tip: Run `tmpl bind ./...` using a [`//go:generate` annotation](https://go.dev/blog/generate) at the root of your project to ensure all of your templates are bound at build time.

`tmpl bind` works at the _package level_ and will generate a single file containing the binding code for all the structs annotated with `//tmpl:bind` in your package.

```go
import (
    _ "embed"
)

var (
    //go:embed login.tmpl.html
    tmplLoginPage string
)

func (*LoginPage) TemplateText() string {
    return tmplLoginPage
}
```

### Compilation

After implementing `TemplateProvider` you're ready to compile your template and use it in your application. 

Currently, it is recommended to compile your template once at program startup using the function `tmpl.MustCompile`:

```go
var (
    LoginTemplate = tmpl.MustCompile(&LoginPage{})
)
```

If any of your template's syntax were to be invalid, the compiler will `panic` on application startup with a detailed error message. 

> If you prefer to avoid panics and handle the error yourself, use the `tmpl.Compile` function variant.

The compiler returns a managed `tmpl.Template` instance. These templates are safe to use from multiple Go routines.

### Rendering

After compilation, you may execute your template by calling one of the generic render functions.

```go
type Template[T TemplateProvider] interface {
	Render(w io.Writer, data T, opts ...RenderOption) error
	RenderToChan(ch chan string, data T, opts ...RenderOption) error
	RenderToString(data T, opts ...RenderOption) (string, error)
}
```

```go
var (
    LoginTemplate = tmpl.MustCompile(&LoginPage{})
)

func main() {
    buf := bytes.Buffer{}
    err := LoginTemplate.Render(&buf, &LoginPage{
        Title:    "Login",
        Username: "",
        Password: "",
    })
    if err != nil {
        panic(err)
    }
	
    fmt.Println(buf.String())
}
```

You can also pass additional options to the render function to customize the behavior of the template.

```go
type RenderOption func(p *RenderProcess)
```

### Template Nesting

One major advantage of using structs to bind templates is that nesting templates is as easy as nesting structs. 

The tmpl compiler knows to recursively look for fields in your dot context struct that also implement the `TemplateProvider` interface. This includes fields that are embedded, slices or pointers.

A good use case for nesting templates is to abstract the document `<head>` of the page into a separate template that can now be shared and reused by other pages:

```html
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }} | torque</title>
    
    {{ range .Scripts -}}
        <script src="{{ . }}"></script>
    {{ end -}}
</head>
```

Again, annotate your dot context struct and run `tmpl bind`:

```go
//tmpl:bind head.tmpl.html
type Head struct {
    Title   string
    Scripts []string
}
```

Now, update the `LoginPage` struct to embed the new `Head` template.

The name of the template is defined using the `tmpl` struct tag. If the tag is not present the field name is used instead.

```go
//tmpl:bind login.tmpl.html
type LoginPage struct {
    Head `tmpl:"head"`
	
    Username string
    Password string
}
```
Embedded templates can be referenced using the built in `{{ template }}` directive. Use the name assigned in the struct tag and ensure to pass the dot context value.
```html
<!DOCTYPE html>
<html lang="en">
{{ template "head" .Head }}
<body>
...
</body>
</html>
```

Finally, update references to `LoginPage` to include the nested template's dot as well.
```go
var (
    LoginTemplate = tmpl.MustCompile(&LoginPage{})
)

func main() {
    buf := bytes.Buffer{}
    err := LoginTemplate.Render(&buf, &LoginPage{
        Head: &Head{
            Title:   "Login",
            Scripts: []string{ "https://unpkg.com/htmx.org@1.9.2" },
        },
        Username: "",
        Password: "",
    })
    if err != nil {
        panic(err)
    }
	
    fmt.Println(buf.String())
}
```

### Targeting

Sometimes you may want to render a nested template. To do this, use the `RenderOption` `WithTarget` in any of the render functions: 

```go
func main() {
    buf := bytes.Buffer{}
    err := LoginTemplate.Render(&buf, &LoginPage{
        Title:    "Login",
        Username: "",
        Password: "",
    }, tmpl.WithTarget("head"))
    if err != nil {
        panic(err)
    }
}
```