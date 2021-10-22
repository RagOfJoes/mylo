# ========= Docker ========= # 
.PHONY: dc-down
dc-down:
	docker-compose down

# ========= Development ========= # 
.PHONY: dev-run
dev-run:
	go run ./cmd/idp

.PHONY: dev-dc-build
dev-dc-build:
	docker-compose build dev

.PHONY: dev-dc-up
dev-dc-up:
	docker-compose up --build dev

.PHONE: dev-dc-run
dev-dc-run:
	docker-compose up dev
