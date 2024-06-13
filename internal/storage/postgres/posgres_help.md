docker run -d --rm --name postgres-storage -e POSTGRES_PASSWORD=pass -e POSTGRES_USER=user -p 127.0.0.1:5432:5432 postgres:16

docker run -v ./internal/storage/postgres/migrations:/migrations --network host migrate/migrate:4 create -ext sql -dir /migrations -seq create_users_table

docker run -v ./internal/storage/postgres/migrations:/migrations --network host migrate/migrate:4 -path=/migrations/ -database postgres://user:pass@localhost:5432/user?sslmode=disable up