# Makefile for QuotaPanel Wails Application

# 

# 
.PHONY: help dev build stop clean install check-deps format update-version

# Default target
help:
	@echo "Available targets:"
	@echo "  dev                       - Start the development server"
	@echo "  format                    - Format and lint-fix all frontend code"
	@echo "  build                     - Build the application"
# 
	@echo "  stop                      - Stop the running QuotaPanel app"
	@echo "  clean                     - Clean build artifacts"
	@echo "  install                   - Install dependencies"
	@echo "  check-deps                - Check if required tools are installed"
	@echo "  update-version            - Update version (usage: make update-version 0.2.5)"
# 

# Check if required tools are installed
check-deps:
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed. Please install Go."; exit 1; }
	@command -v wails >/dev/null 2>&1 || { echo "Wails is not installed. Please install Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest"; exit 1; }
	@command -v pnpm >/dev/null 2>&1 || { echo "pnpm is not installed. Please install pnpm: npm install -g pnpm"; exit 1; }

# Install dependencies
install: check-deps
	cd frontend && pnpm install
	go mod tidy

# Start development server
dev: check-deps
	wails dev

# Format and lint-fix all frontend source code
format:
	cd frontend && pnpm run format
	cd frontend && pnpm run lint:fix

# Build the application
build: check-deps format
	wails build -devtools

# 

# 停止正在运行的 QuotaPanel 应用（含已安装版本与 build/bin 调试实例）
stop:
	@pkill -f "QuotaPanel.app/Contents/MacOS/QuotaPanel" 2>/dev/null || true
	@echo "已停止正在运行的 QuotaPanel（如有）"

# 

# Clean build artifacts
clean:
	rm -rf build/bin
	rm -rf frontend/packages/ui/dist
	rm -rf frontend/packages/quota/dist
	rm -rf frontend/node_modules
	rm -rf frontend/packages/ui/node_modules
	rm -rf frontend/packages/quota/node_modules
	go clean

# 更新版本号（app.go / 前端各 package.json / test/screenshot.ts）
# 用法：make update-version 0.2.5   或   make update-version VERSION=0.2.5
update-version:
	@bash scripts/update-version.sh $(or $(filter-out update-version,$(MAKECMDGOALS)),$(VERSION))

# 将版本号位置参数声明为空目标，避免 make 报 "No rule to make target"
UPDATE_VERSION_ARGS := $(filter-out update-version,$(MAKECMDGOALS))
ifneq ($(filter update-version,$(MAKECMDGOALS)),)
ifneq ($(UPDATE_VERSION_ARGS),)
$(UPDATE_VERSION_ARGS):
	@true
endif
endif

# 
