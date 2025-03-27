build:
	@go build -o bin/mgo ./cmd/web

run: build
	@./bin/mgo -dsn="/Users/markoristic/learn/golang/mgo/db/meinappf.db?_busy_timeout=5000&_journal_mode=WAL"

test:
	@go test ./... -v

deploy: build
	cp bin/mgo /Users/markoristic/bin/
