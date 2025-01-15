1. Creating a migration file
migrate create -seq -ext .sql -dir ./migrations create_movies_table

2. Running a migration file
migrate -path ./migrations -database "postgres://postgres:pass123@localhost/greenlight?sslmode=disable" up

3. Roll back the most recent migration
migrate -path ./migrations -database "postgres://postgres:pass123@localhost/greenlight?sslmode=disable" down 1

4. Migrating to a specific version
migrate -path ./migrations -database "postgres://postgres:pass123@localhost/greenlight?sslmode=disable" goto 1