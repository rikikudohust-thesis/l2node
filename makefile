SHELL: /bin/bash

clean:
	rm -rf stateDB/* syncDB/* L2DB/syncTxDB/* L2DB/txSelDB/*

env:
	docker start 85

sync:
	go run cmd/synchronier/synchronier.go

select:
	go run cmd/selector/selector.go

build:
	go run cmd/batchbuilder/batchbuilder.go

