# voccy

Feedback board API built with Go, chi, and PostgreSQL.

## Requirements

- Go 1.25+
- PostgreSQL
- [golang-migrate](https://github.com/golang-migrate/migrate)

## Quick Start

```bash
git clone https://github.com/rizqishq/voccy.git
cd voccy
cp .env.example .env  # configure your database credentials
make migrateup
make run
```

## API Endpoints

### Boards

| Method | Path | Description |
|--------|------|-------------|
| GET | /boards | List all boards |
| POST | /boards | Create a board |
| GET | /boards/:id | Get board by ID |
| PUT | /boards/:id | Update a board |
| DELETE | /boards/:id | Delete a board |

### Feedbacks

| Method | Path | Description |
|--------|------|-------------|
| GET | /boards/:id/feedbacks | List feedbacks for a board |
| POST | /boards/:id/feedbacks | Create feedback |
| GET | /boards/:id/feedbacks/:feedbackId | Get feedback by ID |
| PATCH | /boards/:id/feedbacks/:feedbackId | Update feedback status |
| DELETE | /boards/:id/feedbacks/:feedbackId | Delete feedback |

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |

## API Collection

Postman collection available at [`voccy.postman_collection.json`](voccy.postman_collection.json). Import it into Postman or any compatible client.

## Development

```bash
make build        # compile binary to bin/
make run          # build and run
make test         # run all tests (requires TEST_DB_URL)
make psql         # open psql shell in docker container
make migrateup    # apply migrations
make migratedown  # rollback migrations
make clean        # remove build artifacts
```

## License

[MIT](LICENSE)
