package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

// Model represents the application state
type Model struct {
	frames          []string
	currentFrame    int
	frameCount      int
	playing         bool
	lastUpdate      time.Time
	width           int
	height          int
	loading         bool
	frameChan       chan string
	audioStarted    bool
	audioPlayer     *AudioPlayer
	audioEnabled    bool
	subtitlesJA     []Subtitle
	subtitlesEN     []Subtitle
	subtitleMode    int // 0: off, 1: JA, 2: EN
	currentSubtitle string
	showControls    bool
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Clean up audio player
			if m.audioPlayer != nil {
				m.audioPlayer.Close()
			}
			return m, tea.Quit
		case " ":
			// Toggle play/pause
			m.playing = !m.playing
			if m.audioPlayer != nil {
				if m.playing {
					if m.audioPlayer.IsPaused() {
						m.audioPlayer.Resume()
					} else {
						m.audioPlayer.Play()
					}
				} else {
					m.audioPlayer.Pause()
				}
			}
			if m.playing {
				return m, tick()
			}
			return m, nil
		case "s":
			// Cycle through subtitle modes
			m.subtitleMode = (m.subtitleMode + 1) % 3
			// Clear current subtitle when changing modes
			m.currentSubtitle = ""
			return m, nil
		case "r":
			// Reset to beginning
			m.currentFrame = 0
			if m.audioPlayer != nil {
				m.audioPlayer.Stop()
				if m.playing {
					m.audioPlayer.Play()
				}
			}
			return m, nil
		}
	case tickMsg:
		if m.playing && m.frameCount > 0 {
			m.currentFrame = (m.currentFrame + 1) % m.frameCount
			m.updateSubtitle()
			// Also check for new frames from background loading
			return m, tea.Batch(tick(), waitForFrame(m.frameChan))
		}
	case framesLoadedMsg:
		m.frames = msg.frames
		m.frameCount = len(msg.frames)
		m.loading = true
		// Auto-start playing when initial frames are loaded
		m.playing = true
		// Initialize audio player only if audio is enabled
		if m.audioEnabled && !m.audioStarted {
			audioPlayer, err := NewAudioPlayer()
			if err != nil {
				fmt.Printf("Warning: Could not initialize audio: %v\n", err)
			} else {
				m.audioPlayer = audioPlayer
				m.audioPlayer.Play()
			}
			m.audioStarted = true
		}
		return m, tea.Batch(tick(), waitForFrame(m.frameChan))

	case frameLoadedMsg:
		// Add frame from background loading
		m.frames = append(m.frames, msg.frame)
		m.frameCount = len(m.frames)
		return m, nil
	case startLoadingMsg:
		m.loading = true
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Start loading frames when we know the terminal size
		if m.frameCount == 0 && !m.loading {
			// Always reserve 3 lines for subtitles
			videoHeight := m.height - 3
			if videoHeight < 1 {
				videoHeight = 1
			}
			return m, tea.Batch(
				loadInitialFrames(m.width, videoHeight),
				listenForFrames(m.frameChan, m.width, videoHeight),
			)
		}
		return m, nil
	}
	return m, nil
}

// View renders the model
func (m Model) View() string {
	if m.frameCount == 0 {
		return "Loading frames...\nPress 'q' to quit, 'space' to play/pause, 'r' to reset, 's' for subtitles"
	}

	var view strings.Builder
	if m.currentFrame < len(m.frames) {
		view.WriteString(m.frames[m.currentFrame])
	} else {
		view.WriteString("No frame to display")
	}

	// Add subtitle or controls to view
	if m.subtitleMode > 0 && m.currentSubtitle != "" {
		view.WriteString("\n\n")

		// Split subtitle into lines and center each line
		lines := strings.Split(m.currentSubtitle, "\n")
		for _, line := range lines {
			// Trim whitespace from the line
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Calculate padding for centering
			padding := (m.width - len(line)) / 2
			if padding < 0 {
				padding = 0
			}

			view.WriteString(strings.Repeat(" ", padding))
			view.WriteString(line)
			view.WriteString("\n")
		}
	} else if m.showControls {
		view.WriteString("\n\n")

		// Controls text
		controls := []string{
			"[space] play/pause | [r] reset | [s] subtitles | [q] quit",
		}

		// Always use dim style
		style := "\033[2m"

		// Display controls with fade
		for _, line := range controls {
			// Calculate padding for centering
			padding := (m.width - len(line)) / 2
			if padding < 0 {
				padding = 0
			}

			view.WriteString(strings.Repeat(" ", padding))
			view.WriteString(style)
			view.WriteString(line)
			view.WriteString("\033[0m") // Reset style
			view.WriteString("\n")
		}
	}

	return view.String()
}

// Messages
type tickMsg time.Time
type framesLoadedMsg struct {
	frames []string
}

type frameLoadedMsg struct {
	frame string
}
type startLoadingMsg struct{}

// Commands
func tick() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(16 * time.Millisecond) // ~60 FPS (1000ms / 60 ≈ 16.67ms)
		return tickMsg(time.Now())
	}
}

func loadInitialFrames(width, height int) tea.Cmd {
	return func() tea.Msg {
		// Load first 30 frames quickly to start playing
		frames := make([]string, 0)
		for i := 1; i <= 30; i++ {
			filename := getFrameFilename(i)
			frame, err := loadFrameAsASCII(filename, width, height)
			if err != nil {
				break
			}
			frames = append(frames, frame)
		}

		return framesLoadedMsg{frames: frames}
	}
}

func listenForFrames(frameChan chan string, width, height int) tea.Cmd {
	return func() tea.Msg {
		// Start background loading of remaining frames
		go loadRemainingFrames(frameChan, width, height)
		return startLoadingMsg{}
	}
}

func waitForFrame(frameChan chan string) tea.Cmd {
	return func() tea.Msg {
		select {
		case frame := <-frameChan:
			return frameLoadedMsg{frame: frame}
		default:
			// No frame available, try again later
			return nil
		}
	}
}

func loadRemainingFrames(frameChan chan string, width, height int) {
	// Get total frame count dynamically
	totalFrames, err := countFrames()
	if err != nil {
		fmt.Printf("Error counting frames: %v\n", err)
		close(frameChan)
		return
	}

	// Load remaining frames starting from frame 31
	for i := 31; i <= totalFrames; i++ {
		filename := getFrameFilename(i)
		frame, err := loadFrameAsASCII(filename, width, height)
		if err != nil {
			fmt.Printf("Error loading frame %d: %v\n", i, err)
			break
		}
		frameChan <- frame
	}
	close(frameChan)
}

// loadFrameAsASCII loads a PNG frame and converts it to ASCII art
func loadFrameAsASCII(filename string, targetWidth, targetHeight int) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return "", err
	}

	// Convert to grayscale if needed
	grayImg, ok := img.(*image.Gray)
	if !ok {
		// Convert to grayscale
		bounds := img.Bounds()
		grayImg = image.NewGray(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				grayImg.Set(x, y, img.At(x, y))
			}
		}
	}

	lines := renderBlocksScaled(grayImg, targetWidth, targetHeight)
	return strings.Join(lines, "\n"), nil
}

func renderBlocksScaled(img image.Image, targetWidth, targetHeight int) []string {
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()

	// Determine if we need to scale down (terminal smaller than source)
	scaleDown := targetWidth < srcW || targetHeight < srcH

	var lines []string

	if scaleDown {
		// For downscaling, use simple nearest neighbor for better performance
		for y := 0; y < targetHeight; y++ {
			var sb strings.Builder
			for x := 0; x < targetWidth; x++ {
				// Map target coordinates to source coordinates
				srcX := (x * srcW) / targetWidth
				srcY := (y * srcH) / targetHeight

				// Clamp to source bounds
				if srcX >= srcW {
					srcX = srcW - 1
				}
				if srcY >= srcH {
					srcY = srcH - 1
				}

				pixel := img.At(srcX, srcY).(color.Gray).Y
				sb.WriteRune(pixelRuneSingle(pixel))
			}
			lines = append(lines, sb.String())
		}
	} else {
		// For upscaling, use bilinear interpolation for smooth results
		for y := 0; y < targetHeight; y++ {
			var sb strings.Builder
			for x := 0; x < targetWidth; x++ {
				// Calculate source coordinates with floating point precision
				srcX := float64(x) * float64(srcW) / float64(targetWidth)
				srcY := float64(y) * float64(srcH) / float64(targetHeight)

				// Get interpolated pixel value
				pixel := bilinearInterpolate(img, srcX, srcY, srcW, srcH)
				sb.WriteRune(pixelRuneSingle(pixel))
			}
			lines = append(lines, sb.String())
		}
	}

	return lines
}

func bilinearInterpolate(img image.Image, x, y float64, maxW, maxH int) uint8 {
	// Get the four surrounding pixels
	x0 := int(x)
	y0 := int(y)
	x1 := x0 + 1
	y1 := y0 + 1

	// Clamp coordinates
	if x1 >= maxW {
		x1 = maxW - 1
	}
	if y1 >= maxH {
		y1 = maxH - 1
	}

	// Get pixel values
	p00 := img.At(x0, y0).(color.Gray).Y
	p01 := img.At(x0, y1).(color.Gray).Y
	p10 := img.At(x1, y0).(color.Gray).Y
	p11 := img.At(x1, y1).(color.Gray).Y

	// Calculate interpolation weights
	fx := x - float64(x0)
	fy := y - float64(y0)

	// Bilinear interpolation
	val := uint8(
		float64(p00)*(1-fx)*(1-fy) +
			float64(p10)*fx*(1-fy) +
			float64(p01)*(1-fx)*fy +
			float64(p11)*fx*fy,
	)

	return val
}

func pixelRuneSingle(pixel uint8) rune {
	// Use more grayscale characters for better detail
	switch {
	case pixel < 32:
		return '█' // full block for very dark
	case pixel < 64:
		return '▓' // dark shade
	case pixel < 96:
		return '▒' // medium shade
	case pixel < 128:
		return '░' // light shade
	case pixel < 192:
		return ' ' // space
	default:
		return ' ' // space for very light
	}
}

func (m *Model) updateSubtitle() {
	// Calculate current video time based on frame number
	// Video starts at frame 1, and subtitles start at ~29 seconds
	// Each frame is ~16.67ms at 60 FPS
	videoTime := time.Duration(m.currentFrame) * (1000 / 60) * time.Millisecond

	// Show controls during intro
	if videoTime < 14600*time.Millisecond {
		m.showControls = true
	} else {
		m.showControls = false
	}

	if m.subtitleMode == 0 {
		m.currentSubtitle = ""
		return
	}

	var subs []Subtitle
	if m.subtitleMode == 1 {
		subs = m.subtitlesJA
	} else {
		subs = m.subtitlesEN
	}

	m.currentSubtitle = ""
	for _, sub := range subs {
		if videoTime >= sub.StartTime && videoTime <= sub.EndTime {
			m.currentSubtitle = sub.Text
			break
		}
	}
}

func initialModel(withAudio bool) Model {
	// Load subtitles synchronously since they're embedded
	ja, errJA := ParseSRT("bad_apple_ja.srt")
	if errJA != nil {
		log.Errorf("could not load japanese subtitles: %v", errJA)
	}
	en, errEN := ParseSRT("bad_apple_en.srt")
	if errEN != nil {
		log.Errorf("could not load english subtitles: %v", errEN)
	}

	return Model{
		frames:       make([]string, 0),
		currentFrame: 0,
		frameCount:   0,
		playing:      false,
		lastUpdate:   time.Now(),
		width:        80, // Default width
		height:       60, // Default height
		loading:      false,
		frameChan:    make(chan string, 100), // Buffer for 100 frames
		audioStarted: false,
		audioPlayer:  nil,
		audioEnabled: withAudio,
		subtitlesJA:  ja,
		subtitlesEN:  en,
		subtitleMode: 0,    // Default to no subtitles
		showControls: true, // Start with controls visible
	}
}

const (
	host = "localhost"
	port = "23234"
)

// args to run in ssh mode or not, and to disable audio
var sshMode bool
var quietMode bool

func main() {
	flag.BoolVar(&sshMode, "ssh", false, "run in ssh mode")
	flag.BoolVar(&quietMode, "q", false, "disable audio")
	flag.Parse()

	// Check if frames directory exists and has frames
	frameCount, err := countFrames()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Please run 'go run -tags=generate .' to generate frames first")
		os.Exit(1)
	}

	if frameCount == 0 {
		fmt.Println("No frames found in frames/ directory")
		fmt.Println("Please run 'go run -tags=generate .' to generate frames first")
		os.Exit(1)
	}

	if sshMode {

		s, err := wish.NewServer(
			wish.WithAddress(net.JoinHostPort(host, port)),
			wish.WithHostKeyPath(".ssh/id_ed25519"),
			wish.WithMiddleware(
				bubbletea.Middleware(teaHandler),
				activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
				logging.Middleware(),
			),
		)
		if err != nil {
			log.Error("Could not start server", "error", err)
		}

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		log.Info("Starting SSH server", "host", host, "port", port)
		go func() {
			if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
				log.Error("Could not start server", "error", err)
				done <- nil
			}
		}()

		<-done
		log.Info("Stopping SSH server")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer func() { cancel() }()
		if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not stop server", "error", err)
		}
	} else {
		p := tea.NewProgram(initialModel(!sshMode && !quietMode), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v", err)
			os.Exit(1)
		}

	}
}

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOption (such as
// tea.WithAltScreen) on a session by session basis.
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	m := initialModel(false)
	pty, _, _ := s.Pty()
	m.width, m.height = pty.Window.Width, pty.Window.Height

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}
