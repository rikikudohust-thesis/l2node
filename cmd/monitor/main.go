package main

import (
  "github.com/rikikudohust-thesis/l2node/internal/pkg/builder"
)

func main()  {
  server, err := builder.NewServer()
  if err != nil {
    panic(err)
  }
  server.Run()
}

