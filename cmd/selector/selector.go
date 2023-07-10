package main

import (
  "github.com/rikikudohust-thesis/l2node/internal/pkg/builder"
)

func main()  {
  selector, err := builder.NewSelector("internal/pkg/config/job_ethereum.yaml")
  if err != nil {
    panic(err)
  }
  selector.Run()
}


