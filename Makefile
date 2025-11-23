.PHONY: build run test clean migrate dev

build:
	docker-compose build

run:
	docker-compose up

test:
	go test ./...

clean:
	docker-compose down -v
	rm -f server

migrate:
	docker-compose run app /server migrate

dev:
	go run ./cmd/server

lint:
	golangci-lint run

deps:
	go mod download
	go mod verify
