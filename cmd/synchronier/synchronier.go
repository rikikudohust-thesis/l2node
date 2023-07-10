package main

import (
  "github.com/rikikudohust-thesis/l2node/internal/pkg/builder"
)

func main()  {
  synchronier, err := builder.NewSynchronior("internal/pkg/config/job_ethereum.yaml")
  if err != nil {
    panic(err)
  }
  synchronier.Run()
}


