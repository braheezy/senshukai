package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("Generating frames from video...")

	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Println("Error: ffmpeg is required but not found in PATH")
		fmt.Println("Please install ffmpeg and try again")
		os.Exit(1)
	}

	// Create frames directory if it doesn't exist
	if err := os.MkdirAll("frames", 0755); err != nil {
		fmt.Printf("Error creating frames directory: %v\n", err)
		os.Exit(1)
	}

	// Use bad_apple.mp4 as the source video
	videoFile := "bad_apple.mp4"
	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		fmt.Printf("Error: video file %s not found\n", videoFile)
		os.Exit(1)
	}

	// Generate frames using ffmpeg
	// Extract frames at 30 FPS, naming them out0001.png, out0002.png, etc.
	cmd := exec.Command("ffmpeg",
		"-i", videoFile,
		"-vf", "scale=640:-1:flags=lanczos,format=gray,fps=60",
		"frames/out%04d.png",
		"-y", // Overwrite existing files
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Running ffmpeg command:", cmd.String())

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running ffmpeg: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Frame generation complete!")
}
