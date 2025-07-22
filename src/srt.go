package main

import (
	"bufio"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"time"
)

//go:embed bad_apple_*.srt
var subtitleFiles embed.FS

// Subtitle represents a single subtitle entry
type Subtitle struct {
	ID        int
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
}

// ParseSRT parses an SRT file and returns a slice of Subtitle objects
func ParseSRT(filename string) ([]Subtitle, error) {
	file, err := subtitleFiles.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open srt file: %w", err)
	}
	defer file.Close()

	var subtitles []Subtitle
	scanner := bufio.NewScanner(file)
	var current Subtitle
	var step int

	for scanner.Scan() {
		line := scanner.Text()

		switch step {
		case 0:
			id, err := strconv.Atoi(line)
			if err == nil {
				current.ID = id
				step++
			}
		case 1:
			parts := strings.Split(line, " --> ")
			if len(parts) == 2 {
				current.StartTime, _ = parseSRTTime(parts[0])
				current.EndTime, _ = parseSRTTime(parts[1])
				step++
			}
		case 2:
			if line == "" {
				subtitles = append(subtitles, current)
				current = Subtitle{}
				step = 0
			} else {
				if current.Text != "" {
					current.Text += "\n"
				}
				current.Text += line
			}
		}
	}
	if current.ID != 0 {
		subtitles = append(subtitles, current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading srt file: %w", err)
	}

	return subtitles, nil
}

// parseSRTTime parses the time format from an SRT file
func parseSRTTime(s string) (time.Duration, error) {
	// 00:00:29,082
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format")
	}

	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	secMs := strings.Split(parts[2], ",")
	sec, _ := strconv.Atoi(secMs[0])
	ms, _ := strconv.Atoi(secMs[1])

	return time.Hour*time.Duration(h) +
		time.Minute*time.Duration(m) +
		time.Second*time.Duration(sec) +
		time.Millisecond*time.Duration(ms), nil
}
