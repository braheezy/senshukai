# senshukai

bad apple in the shell

<video src="https://github.com/user-attachments/assets/e3cb6dd5-35ce-4406-8942-966ba269d3f3" controls="controls" style="max-width: 730px;">
</video>

## Usage

Once running, use these controls:

- **Space** - Play/Pause
- **R** - Reset to beginning
- **Q** or **Ctrl+C** - Quit

## Prerequisites

- Go
- ffmpeg (for frame generation)

### Installing ffmpeg

**macOS:**

```bash
brew install ffmpeg
```

**Ubuntu/Debian:**

```bash
sudo apt update
sudo apt install ffmpeg
```

**Windows:**
Download from [ffmpeg.org](https://ffmpeg.org/download.html) or install via Chocolatey:

```bash
choco install ffmpeg
```

## Build Process

The application uses a two-step build process:

1. **Generate frames** from the video file
2. **Run the application** with the generated frames

### Using Make (Recommended)

```bash
# Generate frames and run the application
make all

# Or step by step:
make generate  # Generate frames from video
make run       # Run the application
```

### Manual Build

```bash
# Generate frames from video
go run cmd/generate/main.go

# Run the application
go run .
```

## Development

```bash
# Clean generated files
make clean

# Install dependencies
make deps

# Build binary
make build
```
