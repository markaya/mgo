build:
	@go build -o bin/mgo ./cmd/web

run: build
	@./bin/mgo

test:
	@go test ./... -v
