include Front.mk

POSTGRESQL_URL := postgres://postgres@localhost:5433/ipphone_db?sslmode=disable
IP := 192.168.1.15:5060

run:
	docker-compose start
	go run cmd/pbx/main.go

tcp-dump:
	sudo tcpdump -i wlo1 -n -A udp port 5060

tcp-dump-with-lo:
	sudo tcpdump -ni any udp port 5060 or udp port 5061 -vv

udp-dump:
	sudo tcpdump -ni any -s 0 -vv udp port 5060

compose-create:
	docker-compose build
	docker-compose create

compose-db-start:
	docker-compose start

db-dev-prepare:
	ENV=dev go run scripts/db_creator/db_creator.go

create-migration:
  # export PATH=$PATH:$(go env GOPATH)/bin
	migrate create -ext sql -dir db/migrations -seq $(NAME)

migration-up:
	migrate -database $(POSTGRESQL_URL) -path db/migrations up

migration-down:
	migrate -database $(POSTGRESQL_URL) -path db/migrations down

migration-force:
	migrate -database $(POSTGRESQL_URL) -path db/migrations force $(V)

connect-db:
	psql --user=postgres --host=localhost --dbname=ipphone_db --port=5433


update-structure:
	./schema.sh

# rebuild deps
.PHONY: tidy
tidy:
	go mod tidy

# run all service in docker
.PHONY: obs-up obs-down
obs-up:
	docker compose up -d --build prometheus grafana postgres sipserver

obs-down:
	docker compose down

## run perf tests

run-perf-test-proxy:
	sipp 192.168.1.15:5060 -sf perf_tests/uac_proxy_cps.xml -s 1001 \
  	-r 200 -l 20000 -m 50000 \
  	-recv_timeout 3000 -timeout 10 \
  	-trace_err -trace_stat -stf proxy_cps_200.csv



run-perf-test-redirect:
	sipp $(IP) -sf perf_tests/uac_redirect_cps_safe.xml -s 1001 \
  	-r 500 -l 20000 -m 50000 \
  	-recv_timeout 3000 -timeout 10 \
  	-trace_err -trace_stat -stf redirect_cps_500.csv

