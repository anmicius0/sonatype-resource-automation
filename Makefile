APP_NAME := sra
MAIN_PKG := ./main.go
BIN_DIR := bin

PLATFORMS := \
	darwin-arm64 \
	linux-amd64 \
	windows-amd64

.PHONY: all clean $(PLATFORMS)

all: $(PLATFORMS)

darwin-arm64:
	@echo "Building macOS arm64..."
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN_PKG)

linux-amd64:
	@echo "Building Linux amd64..."
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_PKG)

windows-amd64:
	@echo "Building Windows amd64..."
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_PKG)

clean:
	rm -rf $(BIN_DIR)
