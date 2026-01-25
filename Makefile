POSTGRESQL_URL := postgres://postgres@localhost:5433/ipphone_db?sslmode=disable

run:
	docker-compose start
	go run cmd/pbx/main.go

tcp-dump:
	sudo tcpdump -i wlo1 -n -A udp port 5060

compose-create:
	docker-compose build
	docker-compose create

compose-db-start:
	docker-compose start

db-dev-prepare:
	ENV=dev go run scripts/db_creator/db_creator.go

create-migration:
	migrate create -ext sql -dir db/migrations -seq $(NAME)

migration-up:
	migrate -database $(POSTGRESQL_URL) -path db/migrations up

migration-down:
	migrate -database $(POSTGRESQL_URL) -path db/migrations down

migration-force:
	migrate -database $(POSTGRESQL_URL) -path db/migrations force $(V)

connect-db:
	psql --user=postgres --host=localhost --dbname=ipphone_db --port=5433
