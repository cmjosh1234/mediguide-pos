include ../scripts/make-platform.mk

-include .env
export

ifeq ($(OS),Windows_NT)
BUILD_VERSION ?= $(strip $(shell type ..\VERSION))
BUILD_REVISION ?= $(shell git rev-parse --verify HEAD 2>NUL || echo unknown)
else
BUILD_VERSION ?= $(shell tr -d '[:space:]' < ../VERSION)
BUILD_REVISION ?= $(shell git rev-parse --verify HEAD 2>/dev/null || printf unknown)
endif
API_LDFLAGS := -X mediguide/internal/buildinfo.Version=$(BUILD_VERSION) -X mediguide/internal/buildinfo.Revision=$(BUILD_REVISION)

.PHONY: run build test tidy migrate-up migrate-down migrate-status seed clinical-tools-check clinical-tools-import clinical-tools-development-activate clinical-tools-retirement-check clinical-tools-retirement-rehearsal-check docker-up docker-down swagger proto

run:
	go run -ldflags "$(API_LDFLAGS)" ./cmd/api

worker:
	go run ./cmd/worker

build:
	go build -ldflags "$(API_LDFLAGS)" -o bin/api$(EXEEXT) ./cmd/api
	go build -o bin/worker$(EXEEXT) ./cmd/worker

test:
	go test ./...

tidy:
	go mod tidy

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

seed:
	go run ./cmd/seed

clinical-tools-check:
	go run ./cmd/clinicaltool-migrate

clinical-tools-import:
	$(if $(strip $(ACTOR_ID)),,$(error ACTOR_ID is required))
	go run ./cmd/clinicaltool-migrate --apply --actor "$(ACTOR_ID)"

clinical-tools-development-activate: export CLINICAL_TOOLS_DEVELOPMENT_ACTIVATION=1
clinical-tools-development-activate:
	go run ./cmd/clinicaltool-development-activate

clinical-tools-retirement-check:
	go run ./cmd/clinicaltool-migrate --retirement-check

clinical-tools-retirement-rehearsal-check:
	go run ./cmd/clinicaltool-rehearsal

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -d cmd/api,internal/handlers,internal/httpx,internal/models,internal/services,internal/clinicaltools,internal/storage -o docs --outputTypes go,json,yaml --generatedTime=false --parseInternal

proto:
	$(BASH) ../scripts/generate_proto.sh

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v
