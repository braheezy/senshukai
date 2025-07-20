package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// countFrames counts the number of frame files in the frames directory
func countFrames() (int, error) {
	entries, err := os.ReadDir("frames")
	if err != nil {
		return 0, fmt.Errorf("error reading frames directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".png") {
			if strings.HasPrefix(entry.Name(), "out") {
				count++
			}
		}
	}

	return count, nil
}

// getFrameFilename returns the filename for a given frame number
func getFrameFilename(frameNum int) string {
	return fmt.Sprintf("frames/out%04d.png", frameNum)
}

// extractFrameNumber extracts the frame number from a filename like "out0001.png"
func extractFrameNumber(filename string) int {
	// Remove "out" prefix and ".png" suffix
	numberStr := strings.TrimPrefix(filename, "out")
	numberStr = strings.TrimSuffix(numberStr, ".png")

	if num, err := strconv.Atoi(numberStr); err == nil {
		return num
	}
	return 0
}
