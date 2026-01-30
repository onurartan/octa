# --- Variables ---
BINARY_NAME := octa
SRC_DIR     := ./cmd/octa
BIN_DIR     := ./bin
CRAFT_SRC   := ./scripts/gocraft.go

# --- OS Detection ---
ifeq ($(OS),Windows_NT)
    EXT := .exe
    MKDIR_CMD := if not exist "$(BIN_DIR)" mkdir "$(BIN_DIR)"
    RM_CMD := del /Q /S
else
    EXT :=
    MKDIR_CMD := mkdir -p $(BIN_DIR)
    RM_CMD := rm -rf
endif

OCTO_BIN    := $(BIN_DIR)/$(BINARY_NAME)$(EXT)
CRAFT_BIN := $(BIN_DIR)/gocraft$(EXT)

.PHONY: all build run clean bench bench-go bench-rust warden craft build-craft help

all: build

# --- Core Commands ---

octa: 
	$(MAKE) icraft
	$(MAKE) irun

run:
	@echo [RUN] Starting application from source...
	@go run $(SRC_DIR)

# Run Binary
irun:
	@echo [RUN] Starting application from source...
	$(OCTO_BIN)

build:
	@echo [BUILD] Compiling $(BINARY_NAME)...
	@$(MKDIR_CMD)
	@go build -ldflags="-s -w" -o $(OCTO_BIN) $(SRC_DIR)


dbseed:
	@echo [RUN] Starting DB Seed...
	@go run ./scripts/dbseed.go

clean:
	@echo [CLEAN] Removing artifacts...
	@go clean
	@rm -rf $(BIN_DIR) 2>NUL || true




# --- GoCraft (Build Engine) ---

craft:
	@echo [CRAFT] Running build engine...
	@go run $(CRAFT_SRC) -n $(BINARY_NAME) -e $(SRC_DIR)

# Build with GoCraft Binary
icraft:
	@echo [CRAFT] Running build engine...
	$(CRAFT_BIN) -n $(BINARY_NAME) -e $(SRC_DIR)

build-craft:
	@echo [BUILD] Compiling GoCraft tool...
	@$(MKDIR_CMD)
	@go build -ldflags="-s -w" -o $(CRAFT_BIN) $(CRAFT_SRC)

# --- Benchmarks ---

bench: bench-rust

bench-go:
	@echo [BENCH] Running internal Go benchmark...
	@go run ./scripts/benchmark.go

bench-rust:
	@echo [BENCH] Running Octa-RustBench (Rust)...
	@cd rust/bench && cargo run --release

# --- Maintenance ---

warden:
	@echo [WARDEN] Running integrity check...
	@cargo run --manifest-path rust/warden/Cargo.toml --release -- --config config.yaml

help:
	@echoUsage:
	@echo  make run          - Run directly (go run)
	@echo  make build        - Compile binary to ./bin
	@echo  make craft        - Run builder script
	@echo  make build-craft  - Compile builder tool
	@echo  make clean        - Clean build artifacts
	@echo  make bench        - Run load tests
	@echo  make warden       - Run integrity tool