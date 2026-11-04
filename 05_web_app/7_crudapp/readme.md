# CRUD app

```sh
go mod tidy
go build -o ./bin/crudapp ./cmd/crudapp
go test -v ./...
```

Optional: vendor dependencies for an offline build.

```sh
go mod vendor
go build -mod=vendor -o ./bin/crudapp ./cmd/crudapp
go test -v -mod=vendor ./...
```
