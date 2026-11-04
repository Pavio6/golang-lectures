# README

https://github.com/golang-standards/project-layout/blob/master/README_ru.md

```sh
go mod init myapp
# Or use the module's import path, for example:
# go mod init github.com/rvasily/myapp
go mod tidy
go build -o ./bin/myapp ./cmd/myapp
go test -v ./...
```

Optional: vendor dependencies for an offline build.

```sh
go mod vendor
go build -mod=vendor -o ./bin/myapp ./cmd/myapp
go test -mod=vendor ./...
```

See also: <https://go.dev/doc/modules/managing-dependencies>.
