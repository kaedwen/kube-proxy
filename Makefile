.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -o service main.go