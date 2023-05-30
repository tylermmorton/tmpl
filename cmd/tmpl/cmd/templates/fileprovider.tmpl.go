func (t *{{ .StructType }}) TemplateText() string {
  byt, err := os.ReadFile("{{.FilePath}}")
  if err != nil {
    panic(err)
  }
  return string(byt)
}

{{ if .UseWatcher }}
func (t *{{.StructType}}) Spawn(signal chan struct{}) {
  ch := make(chan error)

  watcher, err := fsnotify.NewWatcher()
  if err != nil {
    fmt.Printf("[watch] failed to create new watcher: %+v\n", err)
    return
  }
  defer watcher.Close()

  go func(chan error) {
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
  }(ch)

  err = watcher.Add("{{.FilePath}}")
  if err != nil {
    fmt.Printf("[watch] failed to add watcher: %+v\n", err)
    return
  }

  for {
	  select {
	    case err := <- ch:
			  fmt.Printf("[watch] shutting down watcher, encountered error: %+v\n", err)
			  return
    }
  }
}
{{ end }}