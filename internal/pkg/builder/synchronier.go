package builder

import (
	"context"
	"fmt"
	"sync"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/database/statedb"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/service/synchronier"
)

type Synchronier struct {
	cfg  model.JobConfig
	jobs map[string]model.IJob
}

func NewSynchronior(configFile string) (*Synchronier, error) {
	c, err := loadJobConfig(configFile)
	if err != nil {
		return nil, err
	}

	db, err := NewPostgres(&c.Database)
	if err != nil {
		return nil, err
	}

	c.Redis.Prefix += ":" + c.Config.Network
	r, err := New(&c.Redis)
	if err != nil {
		return nil, err
	}

	cfg := statedb.Config{
		Path:    "./syncDB",
		Keep:    128,
		NoLast:  false,
		Type:    statedb.TypeSynchronizer,
		NLevels: 8,
	}
	sdb, err := statedb.NewStateDB(cfg)

	if err != nil {
		panic(err)
	}

	jobs := map[string]model.IJob{
		// model.ServiceL1Tx: synchronier.NewJob(&c.Config, db, r),
		model.ServiceSynchronier: synchronier.NewJob(&c.Config, db, r, sdb),
	}

	return &Synchronier{
		cfg:  c.Config,
		jobs: jobs,
	}, nil
}

func (s *Synchronier) Run() error {
	var wg sync.WaitGroup
	wg.Add(len(s.jobs))

	for _, name := range s.cfg.Jobs {
		ctx := context.Background()
		job, existed := s.jobs[name]
		if !existed {
			fmt.Println("service not supported")
			continue
		}
		go func(c context.Context, j model.IJob) {
			defer wg.Done()
			j.Run(c)
		}(ctx, job)
	}

	wg.Wait()
	return nil
}
