
OS := $(shell uname -s 2>/dev/null || echo Windows_NT)

ifeq ($(OS),Linux)
	BUILD_SCRIPT = build/build.sh
else ifeq ($(OS),Darwin)
	BUILD_SCRIPT = build/build.sh
else ifeq ($(OS),Windows_NT)
	BUILD_SCRIPT = powershell.exe -File build/build.ps1
else
	$(error Unsupported OS: $(OS))
endif

all:
	$(BUILD_SCRIPT)