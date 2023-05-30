# `tmpl`

> ⚠️ `tmpl` is currently experimental.

tmpl is a wrapper around Go's `html/template` package that improves the overall workflow and offers a few helpful utilities for developers building html based applications:
- Compile-time type safety 
- Nested templates and template fragments
- Managed template files: Load from disk in development with hot reloading, embed in binary for production
- Scaffolding CLI for binding templates to Go code
- Template extensibility through compiler plugins
- Static analysis utilities such as parse tree traversal

##  🧰 Installation
```bash
go get github.com/tylermmorton/tmpl
```

For the scaffolding utilities, you can use 
```bash
go install github.com/tylermmorton/tmpl/cmd/tmpl
```

## 🌊 The Workflow

Start by creating a template. You can use any of the standard Go template syntax. For more information on the syntax, see the [Go template documentation](https://golang.org/pkg/text/template/).

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }}</title>
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

To start tying your template to your Go code, declare a struct that represents the "dot context" of the template. The dot context is the value of the "dot" (`{{ . }}`) in Go's template syntax.

```go
type LoginPage struct {
    Title     string
    Username  string
    Password  string
}
```

- In this struct, any _exported_ fields or methods (attached via pointer receiver) will be accessible in your template from the all powerful dot.

### `TemplateProvider`

To turn your struct into a target for the tmpl compiler, your struct type must implement the `TemplateProvider` interface:
```go
type TemplateProvider interface {
    TemplateText() string
}
```

The most straightforward way to accomplish this is to embed the template into your Go program using the `embed` package from the standard library.

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

If you've opted into using the `tmpl` scaffold cli, you can use the `//tmpl:bind` annotation on your dot context struct instead.

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

> Tip: Run the utility using the [`//go:generate` annotation](https://go.dev/blog/generate) in your package

`bind` works at the _package level_ and will generate a single file containing all the binding code for the annotated structs in your package.

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

From here you can compile your template and use it in your application to render the result. It is recommended to compile your template once at program startup by using `MustCompile`:

```go
var (
	LoginTemplate = tmpl.MustCompile(&LoginPage{})
)
```

If any of your template's syntax were to be invalid, the compiler will `panic` on application startup with a detailed error message. 
> If you prefer avoid panics and handle the error yourself, use the `tmpl.Compile` variant.

Once compiled, your template can be executed by calling the generic function `Render`. It takes an `io.Writer` and an instance of the dot context as parameters. 

Compiled templates are safe to use from multiple Go routines.

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

### Template Nesting

One advantage of using structs to bind templates is that nesting templates is as easy as nesting structs. The tmpl compiler will recursively look for fields that implement `TemplateProvider` as well.

A good use case is to abstract the document `<head>` of the Login page into a separate template that can now be shared and reused by other pages:

```html
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }} | torque</title>
    
    {{ range .Scripts -}}
        <script src="{{ . }}"></script>
    {{ end -}}
</head>
```

Again, tie things together with a struct representing the dot context and run the scaffolding utility:

```go
//tmpl:bind head.tmpl.html
type Head struct {
    Title   string
    Scripts []string
}
```

Now, update the `LoginPage` struct to embed the new `Head` template.

```go
//tmpl:bind login.tmpl.html
type LoginPage struct {
    Head `tmpl:"head"`
	
    Username string
    Password string
}
```

Update the login page template to use the nested template named `head`. This can be done using the built-in `template` directive.

>The name of the template is defined using the `tmpl` struct tag. If the tag is not present it can be referenced by its field name.

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
            Title: "Login",
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

### `TemplateWatcher`

It's common to want to be able to see changes to a template without restarting the application. A drawback of embeddinig templates with Go's `embed` package is that it's quite difficult to achieve hot reload functionality.

Instead, implement the `TemplateProvider` interface by reading the file from disk:
```go
import (
    "os"
)

func (*LoginPage) TemplateText() string {
    byt, err := os.ReadFile("~/abs/path/to/login.tmpl.html")
    if err != nil {
        panic(err)
    }
    return string(byt)
}
```

The `TemplateWatcher` interface can be used to signal to tmpl that a compiled template should be recompiled. This operation is also safe across multiple Go routines.
```go
type TemplateWatcher interface {
    Watch(signal chan struct{})
}
```

An example of reloading a template from disk every 5 seconds:
```go
func (*LoginTemplate) Watch(signal chan struct{}) {
    for {
        signal <- struct{}{}
        time.Sleep(time.Second * 5)
    }
}
```

### Hot Reload

When using the `tmpl bind` utility, pass the `--watch` flag to enable the generation of `TemplateWatcher` implementations using the `github.com/fsnotify/fsnotify` package. 

This can be done at a package level with `//go:generate` annotations or an easy way to generate watchers for all packages:

```shell
tmpl bind ./... --watch
```

The `//tmpl:bind` annotation also supports the `--watch` flag and it takes precedence over the value passed to the `tmpl bind` cli
```go
//tmpl:bind login.tmpl.html --watch
type LoginPage struct {
    ...
}
```
