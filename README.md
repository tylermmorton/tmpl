# `tmpl`

tmpl is a wrapper around Go's template engine that improves the overall workflow and adds a few helpful features:
- Compile-time type safety 
- Nested templates
- Managed template files: Load from disk in development with hot reload, embed in binary for production
- Compiler plugins
- Parsing utilities such as parse tree traversal
- Scaffolding tools for embedding templates in Go code

## ⚙️ Installation
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

## 🚀 GoLand Setup
If you're using IntelliJ's GoLand IDE, there's a few powerful features you can use to supercharge your templating workflow. This section outlines some suggested configuration changes you can make:


## Philosophy
Go offers an extremely powerful and versatile templating package as a part of its standard library. One can write anything from HTML based templates to new Go code generators and everything in between. 