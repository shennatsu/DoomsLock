.PHONY: run build test migrate-up migrate-down docker-up docker-down

run:
	go run cmd/api/main.go

build:
	go build -o bin/api cmd/api/main.go

test:
	go test ./... -v

migrate-up:
	@echo "Run migrations with golang-migrate CLI:"
	@echo "migrate -path migrations -database 'postgres://doomslock:doomslock123@localhost:5432/doomslock?sslmode=disable' up"

migrate-down:
	@echo "migrate -path migrations -database 'postgres://doomslock:doomslock123@localhost:5432/doomslock?sslmode=disable' down"

docker-up:
	docker compose up -d postgres redis

docker-down:
	docker compose down

vet:
	go vet ./...

lint:
	golangci-lint run
