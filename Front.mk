APP=pbx
WEB_DIR=web

.PHONY: help
help:
	@echo "make dev      - Go + React (React на :5173, Go на :8080)"
	@echo "make run      - Go (ожидает что web/dist уже собран)"
	@echo "make web-dev  - только React dev-server"
	@echo "make web-build- build React в web/dist"
	@echo "make tidy     - go mod tidy"
	@echo "make db       - postgres через docker compose"


.PHONY: web-install
web-install:
	cd $(WEB_DIR) && npm install

.PHONY: web-dev
web-dev:
	cd $(WEB_DIR) && npm install && npm run dev

run-front:
	cd $(WEB_DIR) && npm run dev

.PHONY: web-build
web-build:
	cd $(WEB_DIR) && npm install && npm run build

