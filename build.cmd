:: set GIN_MODE=release
gox -ldflags="-s -w" -osarch="linux/amd64 windows/amd64" -output="dist/{{.Dir}}_{{.OS}}_{{.Arch}}"