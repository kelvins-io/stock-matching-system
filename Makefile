.PHONY: help up down build rebuild logs ps backend-run frontend-run tidy test clean

COMPOSE ?= docker compose

help:
	@echo "Stock Matching System"
	@echo ""
	@echo "  make up            构建并启动全部服务 (docker compose)"
	@echo "  make down          停止并移除容器"
	@echo "  make build         仅构建镜像"
	@echo "  make rebuild       无缓存重建并启动"
	@echo "  make logs          跟踪全部日志"
	@echo "  make ps            查看服务状态"
	@echo "  make tidy          整理 Go 依赖"
	@echo "  make backend-run   本地运行后端 (需本机 postgres/redis)"
	@echo "  make frontend-run  本地运行前端开发服务器"
	@echo "  make clean         清理本地构建产物"

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

build:
	$(COMPOSE) build

rebuild:
	$(COMPOSE) build --no-cache
	$(COMPOSE) up -d

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

tidy:
	cd backend && go mod tidy

backend-run:
	cd backend && \
		PORT=10020 \
		DATABASE_URL='postgres://stock:stock123@localhost:5432/stock_matching?sslmode=disable' \
		REDIS_ADDR=$${REDIS_ADDR:-localhost:6379} \
		go run ./cmd/server

frontend-run:
	cd frontend && npm install && npm run dev

test:
	cd backend && go test ./...

clean:
	rm -rf frontend/dist frontend/node_modules
	rm -f backend/server /tmp/stock-server
