APP_NAME=pr-reviewer
CMD_PATH=./cmd/app/main.go

.PHONY: run build test lint clean

run:
	go run $(CMD_PATH)

build:
	go build -o ./bin/$(APP_NAME) $(CMD_PATH)

test:
	go test -v -race -timeout 30s ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/