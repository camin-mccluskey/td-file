run:
	go run main.go

build:
	go build -o td-file main.go

test:
	go test ./...

lint:
	go vet ./... 