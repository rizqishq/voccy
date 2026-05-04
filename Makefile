include .env

APP_NAME = voccy
BIN_DIR = bin

.PHONY: build run clean psql migrateup migratedown

build:
	@echo "Building program...."
	@go build -o $(BIN_DIR)/$(APP_NAME) .

run: build
	@echo "Running program...."
	@./$(BIN_DIR)/$(APP_NAME)

clean:
	@echo "Cleaning program"
	@rm -rf $(BIN_DIR)

psql:
	@docker exec -it postgres psql -U $(DB_USER) -d $(DB_NAME)

migrateup:
	@migrate -database $(DB_URL) -path migrations -verbose up

migratedown:
	@migrate -database $(DB_URL) -path migrations -verbose down
