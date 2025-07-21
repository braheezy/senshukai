.PHONY: generate run clean build

# Default target
all: generate build run

# Generate frames from video
generate:
	@echo "Generating frames from video..."
	@which ffmpeg > /dev/null || (echo "Error: ffmpeg is required but not found in PATH" && echo "Please install ffmpeg and try again" && exit 1)
	@mkdir -p frames
	@test -f bad_apple.mp4 || (echo "Error: video file bad_apple.mp4 not found" && exit 1)
	@echo "Running ffmpeg command: ffmpeg -i bad_apple.mp4 -vf scale=640:-1:flags=lanczos,format=gray,fps=60 frames/out%04d.png -y"
	@ffmpeg -i bad_apple.mp4 -vf "scale=640:-1:flags=lanczos,format=gray,fps=60" frames/out%04d.png -y
	@echo "Frame generation complete!"

# Build the application
build:
	@echo "Building application..."
	@cd ./go/ && go build -o ../senshukai . && cd ..

# Run the application
run: build
	@echo "Running application..."
	./senshukai

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
