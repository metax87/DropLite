SHELL := /bin/bash

.PHONY: bootstrap dev dev-backend dev-frontend test lint

bootstrap:
	@echo "→ Installing backend dependencies"
	@cd backend && go mod tidy
	@echo "→ Installing frontend dependencies"
	@cd frontend && pnpm install

dev: dev-backend dev-frontend

dev-backend:
	@cd backend && go run ./cmd/server

dev-frontend:
	@cd frontend && pnpm run dev

test:
	@cd backend && go test ./...
	@cd frontend && pnpm run test

lint:
	@cd backend && golangci-lint run ./...
	@cd frontend && pnpm run lint
