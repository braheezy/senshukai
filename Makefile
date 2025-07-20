.PHONY: generate run clean build

# Default target
all: generate run

# Generate frames from video
generate:
	@echo "Generating frames from video..."
	go run cmd/generate/main.go

# Run the application
run:
	@echo "Running application..."
	go run ./go/

# Build the application
build:
	@echo "Building application..."
	@cd ./go/ && go build -o ../senshukai . && cd ..

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf frames/
	rm -f senshukai

# Check if ffmpeg is installed
check-ffmpeg:
	@which ffmpeg > /dev/null || (echo "Error: ffmpeg is required but not found. Please install ffmpeg first." && exit 1)

# Generate frames with ffmpeg check
generate-frames: check-ffmpeg generate

# Help target
help:
	@echo "Available targets:"
	@echo "  generate      - Generate frames from video (requires ffmpeg)"
	@echo "  run          - Run the application"
	@echo "  build        - Build the application"
	@echo "  clean        - Remove generated files"
	@echo "  check-ffmpeg - Check if ffmpeg is installed"
	@echo "  all          - Generate frames and run application"
