package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

// AudioPlayer manages audio playback with pause/resume functionality
type AudioPlayer struct {
	player     *oto.Player
	context    *oto.Context
	decoder    *mp3.Decoder
	file       *os.File
	playing    bool
	paused     bool
	mu         sync.Mutex
	stopChan   chan struct{}
	resumeChan chan struct{}
}

// NewAudioPlayer creates a new audio player
func NewAudioPlayer() (*AudioPlayer, error) {
	// Open the MP3 file
	file, err := os.Open("bad_apple.mp3")
	if err != nil {
		return nil, fmt.Errorf("error opening audio file: %w", err)
	}

	// Decode the MP3
	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("error decoding MP3: %w", err)
	}

	// Initialize oto
	otoCtx, readyChan, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("error initializing oto: %w", err)
	}

	// Wait for the audio context to be ready
	<-readyChan

	// Create a player
	player := otoCtx.NewPlayer(decoder)

	return &AudioPlayer{
		player:     player,
		context:    otoCtx,
		decoder:    decoder,
		file:       file,
		playing:    false,
		paused:     false,
		stopChan:   make(chan struct{}),
		resumeChan: make(chan struct{}),
	}, nil
}

// Play starts audio playback
func (ap *AudioPlayer) Play() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.playing {
		return
	}

	ap.playing = true
	ap.paused = false
	ap.player.Play()

	// Start playback monitoring in a goroutine
	go ap.monitorPlayback()
}

// Pause pauses audio playback
func (ap *AudioPlayer) Pause() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if !ap.playing || ap.paused {
		return
	}

	ap.paused = true
	ap.player.Pause()
}

// Resume resumes audio playback
func (ap *AudioPlayer) Resume() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if !ap.playing || !ap.paused {
		return
	}

	ap.paused = false
	ap.player.Play()
	ap.resumeChan <- struct{}{}
}

// Stop stops audio playback and resets to beginning
func (ap *AudioPlayer) Stop() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if !ap.playing {
		return
	}

	ap.playing = false
	ap.paused = false
	ap.stopChan <- struct{}{}

	// Close current player and create a new one
	ap.player.Close()

	// Reset decoder to beginning
	ap.file.Seek(0, 0)
	var err error
	ap.decoder, err = mp3.NewDecoder(ap.file)
	if err != nil {
		return
	}
	ap.player = ap.context.NewPlayer(ap.decoder)
}

// IsPlaying returns true if audio is currently playing
func (ap *AudioPlayer) IsPlaying() bool {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	return ap.playing && !ap.paused
}

// IsPaused returns true if audio is paused
func (ap *AudioPlayer) IsPaused() bool {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	return ap.playing && ap.paused
}

// Close cleans up resources
func (ap *AudioPlayer) Close() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.playing {
		ap.stopChan <- struct{}{}
	}

	if ap.player != nil {
		ap.player.Close()
	}
	if ap.file != nil {
		ap.file.Close()
	}
	// Note: oto.Context doesn't have a Close method, it's managed by the library
}

// monitorPlayback monitors the audio playback and handles completion
func (ap *AudioPlayer) monitorPlayback() {
	for {
		select {
		case <-ap.stopChan:
			return
		case <-ap.resumeChan:
			// Continue monitoring after resume
		default:
			ap.mu.Lock()
			if !ap.playing || ap.paused {
				ap.mu.Unlock()
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if !ap.player.IsPlaying() {
				ap.playing = false
				ap.mu.Unlock()
				return
			}
			ap.mu.Unlock()

			time.Sleep(100 * time.Millisecond)
		}
	}
}
