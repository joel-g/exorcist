.PHONY: build run clean test help

build:
	@echo "Building exorcist..."
	@go build -o exorcist ./cmd/exorcist

run: build
	@echo "Running exorcist..."
	@./exorcist

clean:
	@echo "Cleaning..."
	@rm -f exorcist
	@rm -f *.db *.db-shm *.db-wal

test:
	@echo "Running tests..."
	@go test -v ./...

help:
	@echo "Available commands:"
	@echo "  make build  - Build the exorcist binary"
	@echo "  make run    - Build and run exorcist"
	@echo "  make clean  - Remove binary and database files"
	@echo "  make test   - Run tests"
	@echo "  make help   - Show this help message"
