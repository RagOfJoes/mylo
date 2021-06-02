# ========= Docker ========= # 
.PHONY: dc-down
dc-down:
	docker-compose down

# ========= Development ========= # 
.PHONY: dev-build
dev-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags='-w -s -extldflags "-static"' -a \
  -o ./build/dev cmd/idp/main.go

.PHONY: dev-run
dev-run:
	./build/dev

.PHONY: dev-dc-build
dev-dc-build:
	docker-compose build dev

.PHONY: dev-dc-up
dev-dc-up:
	docker-compose up --build dev

.PHONE: dev-dc-run
dev-dc-run:
	docker-compose up dev

.PHONY: dev-dc-down
dev-dc-down:
	docker-compose down

# ========= Production ========= # 
.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags='-w -s -extldflags "-static"' -a \
  -o app cmd/idp/main.go

.PHONY: dc-build
dc-build:
	docker-compose build prod

.PHONY: dc-up
dc-up:
	docker-compose up --build prod

.PHONY: dc-run
dc-run:
	docker-compose run prod
