func (t *{{ .StructType }}) TemplateText() string {
  byt, err := os.ReadFile("{{.FilePath}}")
  if err != nil {
    panic(err)
  }
  return string(byt)
}

{{ if .UseWatcher }}
func (t *{{.StructType}}) WatchSignal(signal chan struct{}, ch chan error) {
  watcher, err := fsnotify.NewWatcher()
  if err != nil {
    ch <- err
    return
  }
  defer watcher.Close()

  // Recover any panics from reading the file
  // and pass it along the given error channel
  defer func(ch chan error) {
    if err, ok := recover().(error); ok && err != nil {
      ch <- err
    }
  }(ch)

  go func() {
    for {
      select {
      case event, ok := <-watcher.Events:
        if !ok {
          return
        }
        if event.Has(fsnotify.Write) {
          signal <- struct{}{}
        }
      case err, ok := <-watcher.Errors:
        if !ok {
          return
        }
        ch <- err
      }
    }
  }()

  err = watcher.Add("{{.FilePath}}")
  if err != nil {
    ch <- err
    return
  }

  // Block goroutine forever so the watcher doesn't get gc'd
  <-make(chan struct{})
}
{{ end }}