include scripts/make-platform.mk

BACKEND_DIR := backend
AI_WORKER_DIR := ai-worker
COMPOSE_FILE := infra/docker-compose.yml
DEV_COMPOSE_FILE := infra/docker-compose.dev.yml
DEV_ENV_FILE := infra/development.env
PRODUCTION_ENV_FILE ?= infra/production.env
DOCKER_COMPOSE := docker compose --env-file "$(DEV_ENV_FILE)" -f "$(COMPOSE_FILE)" -f "$(DEV_COMPOSE_FILE)"
PRODUCTION_COMPOSE := docker compose --env-file "$(PRODUCTION_ENV_FILE)" -f "$(COMPOSE_FILE)"
AI_REQUIREMENTS_FILE := $(AI_WORKER_DIR)/requirements.txt
AI_REQUIREMENTS_STAMP := $(AI_WORKER_DIR)/.requirements.sha256
# Native Windows does not provide a POSIX shell or sudo. WSL uses the
# existing Linux recipes; native GNU Make delegates stack operations to PS.
WINDOWS_DEV_STACK := powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/dev-stack.ps1

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo Available targets:
	@echo up               Build and start the full development stack
	@echo down             Stop the development stack and preserve data
	@echo reset            Stop development and remove its volumes
	@echo build            Build the development stack
	@echo ps               Show development stack status
	@echo logs             Tail development stack logs
	@echo guidelines-logs  Tail the integrated guidelines service
	@echo config           Render the merged development configuration
	@echo env-check        Verify development, staging, production, and Compose variable parity
	@echo prod-up          Pull, migrate, and start production with health waiting
	@echo prod-migrate     Apply migrations with the configured production API image
	@echo prod-down        Stop the production stack and preserve data
	@echo prod-build       Build production images
	@echo prod-pull        Pull production images
	@echo prod-ps          Show production stack status
	@echo prod-logs        Tail production stack logs
	@echo prod-config      Render the production configuration
	@echo test             Run backend and ai-worker tests
	@echo backend-test     Run backend Go tests
	@echo backend-build    Build backend binaries
	@echo backend-run      Run backend API locally
	@echo backend-worker   Run backend Go worker locally
	@echo swagger          Generate backend Swagger JSON/YAML docs
	@echo contracts        Regenerate Go, TypeScript, and Dart API contracts
	@echo contracts-check  Verify committed API contracts have no drift
	@echo release-prepare  Synchronize all versions from RELEASE_TAG
	@echo release-patch    Calculate and prepare the next patch release
	@echo release-minor    Calculate and prepare the next minor release
	@echo release-major    Calculate and prepare the next major release
	@echo release-check    Validate RELEASE_TAG metadata, Git state, and Compose
	@echo migrate-up       Apply backend migrations
	@echo migrate-down     Roll back backend migrations
	@echo migrate-status   Show backend migration status
	@echo seed             Seed backend data
	@echo clinical-tools-check  Verify legacy tool source checksums and conversion envelopes
	@echo clinical-tools-import Import ready clinical-tool conversions as reviewed drafts
	@echo ai-test          Run ai-worker tests
	@echo ai-api           Run ai-worker FastAPI locally
	@echo ai-worker        Run ai-worker loop locally

.PHONY: up
up:
ifeq ($(OS),Windows_NT)
	$(WINDOWS_DEV_STACK) -Action up
else
	$(DOCKER_COMPOSE) up -d --build
endif

.PHONY: down
down:
ifeq ($(OS),Windows_NT)
	$(WINDOWS_DEV_STACK) -Action down
else
	$(DOCKER_COMPOSE) down --remove-orphans
endif

.PHONY: reset
reset:
ifeq ($(OS),Windows_NT)
	$(WINDOWS_DEV_STACK) -Action reset -ConfirmReset
else
	$(DOCKER_COMPOSE) down --volumes --remove-orphans
endif

.PHONY: build
build:
ifeq ($(OS),Windows_NT)
	$(WINDOWS_DEV_STACK) -Action build
else
	$(DOCKER_COMPOSE) build
endif

.PHONY: ps
ps:
ifeq ($(OS),Windows_NT)
	$(WINDOWS_DEV_STACK) -Action ps
else
	$(DOCKER_COMPOSE) ps
endif

.PHONY: logs
logs:
ifeq ($(OS),Windows_NT)
	$(WINDOWS_DEV_STACK) -Action logs
else
	$(DOCKER_COMPOSE) logs -f
endif

.PHONY: config
config:
ifeq ($(OS),Windows_NT)
	$(WINDOWS_DEV_STACK) -Action config
else
	$(DOCKER_COMPOSE) config
endif

.PHONY: env-check
env-check:
	$(BASH) scripts/check-infra-env-parity.sh

.PHONY: guidelines-logs
guidelines-logs:
ifeq ($(OS),Windows_NT)
	$(WINDOWS_DEV_STACK) -Action guidelines-logs
else
	$(DOCKER_COMPOSE) logs -f guidelines
endif

.PHONY: prod-up
prod-up: production-env-check
	$(PRODUCTION_COMPOSE) pull
	$(PRODUCTION_COMPOSE) run --rm api /app/migrate up
	$(PRODUCTION_COMPOSE) up --no-build -d --wait --wait-timeout 600

.PHONY: prod-migrate
prod-migrate: production-env-check
	$(PRODUCTION_COMPOSE) run --rm api /app/migrate up

.PHONY: prod-down
prod-down: production-env-check
	$(PRODUCTION_COMPOSE) down --remove-orphans

.PHONY: prod-build
prod-build: production-env-check
	$(PRODUCTION_COMPOSE) build

.PHONY: prod-pull
prod-pull: production-env-check
	$(PRODUCTION_COMPOSE) pull

.PHONY: prod-ps
prod-ps: production-env-check
	$(PRODUCTION_COMPOSE) ps

.PHONY: prod-logs
prod-logs: production-env-check
	$(PRODUCTION_COMPOSE) logs -f

.PHONY: prod-config
prod-config: production-env-check
	$(PRODUCTION_COMPOSE) config

.PHONY: production-env-check
production-env-check:
	$(if $(wildcard $(subst $(space),\$(space),$(PRODUCTION_ENV_FILE))),,$(error Missing $(PRODUCTION_ENV_FILE). Copy infra/production.env.example and replace every placeholder))

.PHONY: test
test: backend-test ai-test

.PHONY: backend-test
backend-test:
	$(MAKE) -C $(BACKEND_DIR) test

.PHONY: backend-build
backend-build:
	$(MAKE) -C $(BACKEND_DIR) build

.PHONY: backend-run
backend-run:
	$(MAKE) -C $(BACKEND_DIR) run

.PHONY: backend-worker
backend-worker:
	$(MAKE) -C $(BACKEND_DIR) worker

.PHONY: swagger
swagger:
	$(MAKE) -C $(BACKEND_DIR) swagger

.PHONY: contracts
contracts:
	$(BASH) scripts/generate-contracts.sh

.PHONY: contracts-check
contracts-check:
	$(BASH) scripts/check-generated-contracts.sh

.PHONY: clinical-tools-check clinical-tools-import clinical-tools-development-activate clinical-tools-development-schema-up clinical-tools-retirement-check clinical-tools-retirement-rehearsal
clinical-tools-check:
	$(MAKE) -C $(BACKEND_DIR) clinical-tools-check

clinical-tools-import:
	$(MAKE) -C $(BACKEND_DIR) clinical-tools-import ACTOR_ID="$(ACTOR_ID)"

clinical-tools-development-activate:
	$(BASH) scripts/clinical-tools-development-activate.sh

clinical-tools-development-schema-up:
	$(BASH) scripts/clinical-tools-development-schema-up.sh

clinical-tools-retirement-check:
	$(MAKE) -C $(BACKEND_DIR) clinical-tools-retirement-check

clinical-tools-retirement-rehearsal:
	$(BASH) scripts/clinical-tools-retirement-rehearsal.sh

RELEASE_TAG ?=
MOBILE_BUILD_NUMBER ?=

.PHONY: release-prepare
release-prepare:
	node scripts/prepare-release.js "$(RELEASE_TAG)" $(if $(MOBILE_BUILD_NUMBER),--build-number "$(MOBILE_BUILD_NUMBER)",)

.PHONY: release-patch release-minor release-major
release-patch:
	node scripts/prepare-release.js patch $(if $(MOBILE_BUILD_NUMBER),--build-number "$(MOBILE_BUILD_NUMBER)",)

release-minor:
	node scripts/prepare-release.js minor $(if $(MOBILE_BUILD_NUMBER),--build-number "$(MOBILE_BUILD_NUMBER)",)

release-major:
	node scripts/prepare-release.js major $(if $(MOBILE_BUILD_NUMBER),--build-number "$(MOBILE_BUILD_NUMBER)",)

.PHONY: release-check
release-check:
	$(BASH) scripts/check-release-readiness.sh "$(RELEASE_TAG)"

.PHONY: migrate-up
migrate-up:
	$(MAKE) -C $(BACKEND_DIR) migrate-up

.PHONY: migrate-down
migrate-down:
	$(MAKE) -C $(BACKEND_DIR) migrate-down

.PHONY: migrate-status
migrate-status:
	$(MAKE) -C $(BACKEND_DIR) migrate-status

.PHONY: seed
seed:
	$(MAKE) -C $(BACKEND_DIR) seed

.PHONY: ai-test
ai-test: ai-deps
	cd "$(AI_WORKER_DIR)" && $(PYTHON) -m pytest

.PHONY: ai-deps
ai-deps:
	$(PYTHON) scripts/install-ai-deps.py "$(AI_REQUIREMENTS_FILE)" "$(AI_REQUIREMENTS_STAMP)"

.PHONY: ai-api
ai-api: ai-deps
	cd "$(AI_WORKER_DIR)" && $(PYTHON) -m uvicorn app.main:app --reload --host 0.0.0.0 --port 8090

.PHONY: ai-worker
ai-worker: ai-deps
	cd "$(AI_WORKER_DIR)" && $(PYTHON) -m app.worker

.PHONY: run-cfdp-ios-simulator
run-cfdp-ios-simulator:
ifeq ($(OS),Windows_NT)
	$(error The iOS simulator requires macOS. Use make run-cfdp-android on Windows)
else
	cd user_app && flutter run --flavor development --target lib/main_development.dart --dart-define=MEDIGUIDE_API_BASE_URL=http://127.0.0.1:8080
endif

ANDROID_DEVICE ?= emulator-5554
ANDROID_API_BASE_URL ?= http://localhost:8080

.PHONY: run-cfdp-android
run-cfdp-android:
	cd user_app && flutter run -d "$(ANDROID_DEVICE)" --flavor development --target lib/main_development.dart --dart-define=MEDIGUIDE_API_BASE_URL=$(ANDROID_API_BASE_URL)
