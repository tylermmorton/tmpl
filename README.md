# `tmpl`

tmpl is a wrapper around Go's template engine that improves the overall workflow and offers a few helpful utilities:
- Compile-time type safety 
- Nested templates
- Managed template files: Load from disk in development with hot reload, embed in binary for production
- Template extensibility through compiler plugins
- Static analysis utilities such as parse tree traversal
- Scaffolding tools for embedding templates in Go code

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

To tie your template to your Go code, declare a struct that represents the 'dot context' of your template. If you're using the scaffold utility, annotate your struct with the `tmpl:bind` declaration. 

```go
package main 

//go:generate tmpl bind -o tmpl.gen.go

//tmpl:bind login.tmpl.html
type LoginPage struct {
    Title     string
    Username  string
    Password  string
}
```

Executing the bind utility will generate a few things for you. Here's an example of the template bound via `go:embed`. This code usually lives in a separate file in your package named `tmpl.gen.go`

```go
package main

import (
	_ "embed"
)

//go:embed login.tmpl.html
var tmplLoginPageTemplateText string

// TemplateText implements the tmpl.TemplateTextProvider interface
func (t *LoginPage) TemplateText() string {
	return tmplLoginPageTemplateText
}
```

From here you can compile your template and use it in your application to render the result:

```go
package main

import (
	"bytes"
	"fmt"
	
	"github.com/tylermmorton/tmpl"
)

//go:generate tmpl bind -o tmpl.gen.go

//tmpl:bind login.tmpl.html
type LoginPage struct {
    Title    string
    Username string
    Password string
}

// Compile your templates when the program initializes
var (
    // LoginTemplate can be used to render login.tmpl.html
    LoginTemplate = tmpl.Compile(LoginPage{})
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

To be fair, this is a lot of work to render one template. The real power of `tmpl` comes from the ability to nest templates and use them in multiple places. Let's abstract the document `<head>` of our Login page into a separate template to be reused by other pages:

```html
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }}</title>
</head>
```

Again, tie things together with a struct representing the dot context and run the bind utility:

```go
package main

//go:generate tmpl bind -o tmpl.gen.go

//tmpl:bind head.tmpl.html
type Head struct {
    Title string
}
```

Nesting templates in Go is as easy as embedding one template struct in another:

```go
package main

//tmpl:bind head.tmpl.html
type Head struct {
    Title string
}

//tmpl:bind login.tmpl.html
type LoginPage struct {
    Head `tmpl:"head"`
	
    Username string
    Password string
}
```

Now update your login page template to use the new template named `head`:

```html
<!DOCTYPE html>
<html lang="en">
{{ template "head" .Head }}
<body>
...
</body>
</html>
```

When you call `tmpl.Compile` on a template struct, it will recursively compile all the nested templates. You can then render the top-level template and all of its nested templates will be rendered as well, as long as they are referenced by the Go template's native `template` directive. Here is the updated code:

```go
package main

import (
	"bytes"
	"fmt"
	
	"github.com/tylermmorton/tmpl"
)

//go:generate tmpl bind -o tmpl.gen.go

//tmpl:bind head.tmpl.html
type Head struct {
	Title string
}

//tmpl:bind login.tmpl.html
type LoginPage struct {
	Head `tmpl:"head"`

	Username string
	Password string
}

// Compile your templates when the program initializes
var (
    // LoginTemplate can be used to render login.tmpl.html
    LoginTemplate = tmpl.Compile(LoginPage{})
)

func main() {
    buf := bytes.Buffer{}
    err := LoginTemplate.Render(&buf, &LoginPage{
        Head: &Head{
            Title: "Login",
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

## ⚙️ Configuration

`tmpl` uses environment variables to configure the behavior of the package. The following variables are available:

### Bind Mode
```dotenv
TMPL_BIND_MODE=file|watch|embed
```
The bind mode specifies how the `tmpl bind` utility should generate the code used to bind your template files to your Go program.

You can override this on a per-template basis by passing the `--mode` flag on the bind annotation:
```go
package main 

//go:generate tmpl bind

//tmpl:bind login.tmpl.html --mode=embed
type LoginPage struct {}
```

### Bind Go Type
```dotenv
TMPL_BIND_GOTYPE=true|false
```

When set to `true` (default) the `tmpl bind` utility will insert a Go type annotation on the first line of your template. This pattern is supported by GoLand and enables code completion and other IDE features.

Here is an example of what gets generated. This has no effect on your template output:
```html
{{-/* gotype:github.com/tylermmorton/tmpl.LoginPage */-}}
<!DOCTYPE html>
<html lang="en">
...
```