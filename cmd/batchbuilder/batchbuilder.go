package main

import (
  "github.com/rikikudohust-thesis/l2node/internal/pkg/builder"
)

func main()  {
  batchbuilder, err := builder.NewBatchBuilder("internal/pkg/config/job_ethereum.yaml")
  if err != nil {
    panic(err)
  }
  batchbuilder.Run()
}


