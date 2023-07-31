# l2node

The Privacy Layer 2 - The privacy and scaling for ethereum

### Prerequires

- Docker & docker-compose
- go >= 1.20

### Setup db and redis

#### Generate database and redis

```
docker compose up -d
```

#### Create table and index for database

- Run sql command in internal/pkg/database/initdb/readme.md

### Install package

```go
go mod tidy
```

### Generate zkey, wasm file

There are two way to generate zkey and wasm file:

- Read tutorial in L2-Circuit
- Download zkey and wasm file from drive [https://drive.google.com/drive/folders/1sOm85KzfNm3GNuceQ_myML2pu4jUwsJ-?usp=sharing]

### Copy zkey and wasm file to internal/pkg/zkey | internal/pkg/wasm

### Run scanner

```go
go run cmd/synchronier/synchronier.go
```

### Run Executor

```go
go run cmd/selector/selector.go

go run cmd/batchbuilder/batchbuilder.go
```

### Run monitor

```go
go run cmd/monitor/main.go
```
