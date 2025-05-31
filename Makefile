run:
	go run main.go

build:
	go build -o td-file main.go

test:
	go test ./...

lint:
	go vet ./... 

todo-test:
	go run main.go -todo-file test-todos.md