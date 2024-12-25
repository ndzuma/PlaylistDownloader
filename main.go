package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kkdai/youtube/v2"
	"github.com/sqweek/dialog"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type Song struct {
	Name   string
	Artist string
	id     string
}

type Playlist struct {
	Id    string
	Name  string
	Songs []Song
}

type DownloadError struct {
	Song Song
	Err  error
}

func (de DownloadError) Error() string {
	return fmt.Sprintf("Failed to download %s by %s: %v", de.Song.Name, de.Song.Artist, de.Err)
}

func main() {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}

	log.SetOutput(io.Discard)
	fmt.Println("Welcome to the playlist downloader!")
	fmt.Println("Enter the playlist URL (Make sure it's public): ")
	var playlistURL string
	fmt.Scanln(&playlistURL)
	outputDir, err := dialog.Directory().Title("Select Output Directory").Browse()
	if err != nil {
		fmt.Printf("Error selecting directory: %v\n", err)
		return
	}

	// testing extractPlaylistInfo
	err = processPlaylist(playlistURL, outputDir)
	if err != nil {
		fmt.Printf("Error processing playlist: %v\n", err)
	}
	fmt.Println("Press enter to exit...")
	fmt.Scanln()
}

func processPlaylist(url, outputDir string) error {
	// get playlist songs
	playlist, err := extractYouTubePlaylist(url)
	if err != nil {
		return fmt.Errorf("failed to extract playlist info: %w", err)
	}

	// create output directory
	fmt.Println("Choose folder name (keep blank for default name): ")
	var folderName string
	fmt.Scanln(&folderName)
	if folderName == "" {
		folderName = "playlist_" + playlist.Id
	}
	outputDir = filepath.Join(outputDir, folderName)

	// main download
	concurrentErrors := concurrentDownload(playlist, outputDir)
	if len(concurrentErrors) > 0 {
		fmt.Println("\nConcurrent download errors:")
		for _, err := range concurrentErrors {
			fmt.Println(err)
		}
	} else {
		fmt.Println("\nAll downloads completed successfully!")
	}

	return nil
}

func speedTest(outputDir string, playlist Playlist) {
	// Measure concurrent download time
	concurrentStart := time.Now()
	cname := outputDir + "_concurrent"
	concurrentErrors := concurrentDownload(playlist, cname)
	concurrentDuration := time.Since(concurrentStart)

	// Measure sequential download time
	sequentialStart := time.Now()
	sname := outputDir + "_sequential"
	sequentialErrors := sequentialDownload(playlist, sname)
	sequentialDuration := time.Since(sequentialStart)

	fmt.Printf("Concurrent download time: %v\n", concurrentDuration)
	fmt.Printf("Sequential download time: %v\n", sequentialDuration)
	fmt.Printf("Speed improvement: %.2f%%\n", (float64(sequentialDuration-concurrentDuration)/float64(sequentialDuration))*100)

	// Display errors only if they exist
	if len(concurrentErrors) > 0 {
		fmt.Println("\nConcurrent download errors:")
		for _, err := range concurrentErrors {
			fmt.Println(err)
		}
	}

	if len(sequentialErrors) > 0 {
		fmt.Println("\nSequential download errors:")
		for _, err := range sequentialErrors {
			fmt.Println(err)
		}
	}

	if len(concurrentErrors) == 0 && len(sequentialErrors) == 0 {
		fmt.Println("\nAll downloads completed successfully!")
	}
}

func concurrentDownload(playlist Playlist, outputDir string) []DownloadError {
	var wg sync.WaitGroup
	errorChan := make(chan DownloadError, len(playlist.Songs))
	semaphore := make(chan struct{}, 5)

	for _, song := range playlist.Songs {
		wg.Add(1)
		go func(s Song) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", s.id)
			err := downloadYtVideoWithRetry(videoURL, outputDir, 3)
			if err != nil {
				errorChan <- DownloadError{Song: s, Err: err}
			}
		}(song)
	}

	wg.Wait()
	close(errorChan)

	var errors []DownloadError
	for err := range errorChan {
		errors = append(errors, err)
	}

	return errors
}

func sequentialDownload(playlist Playlist, outputDir string) []DownloadError {
	var errors []DownloadError
	for _, song := range playlist.Songs {
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", song.id)
		err := downloadYtVideoWithRetry(videoURL, outputDir, 3)
		if err != nil {
			errors = append(errors, DownloadError{Song: song, Err: err})
		}
	}
	return errors
}

func extractYouTubePlaylist(link string) (Playlist, error) {
	client := youtube.Client{}
	p := Playlist{
		Id:    "",
		Name:  "",
		Songs: []Song{},
	}

	// Extract playlist ID from link
	playlistID, err := extractYouTubePlaylistID(link)
	if err != nil {
		return Playlist{}, err
	}
	p.Id = playlistID

	// Fetch playlist videos
	playlist, err := client.GetPlaylist(playlistID)
	if err != nil {
		return Playlist{}, err
	}
	p.Name = playlist.Title

	//var songs []Song
	for _, video := range playlist.Videos {
		p.Songs = append(p.Songs, Song{
			Name:   video.Title,
			Artist: video.Author,
			id:     video.ID,
		})
	}

	return p, nil
}

func extractYouTubePlaylistID(link string) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Check if the host is youtube.com or youtu.be
	if !strings.Contains(parsedURL.Host, "youtube.com") && !strings.Contains(parsedURL.Host, "youtu.be") {
		return "", fmt.Errorf("invalid YouTube URL")
	}

	// First, check if the list parameter is in the query string
	queryParams := parsedURL.Query()
	if listID := queryParams.Get("list"); listID != "" {
		return listID, nil
	}

	// If not in query, check if it's in the path
	pathParts := strings.Split(parsedURL.Path, "/")
	for i, part := range pathParts {
		if part == "playlist" && i+1 < len(pathParts) {
			return pathParts[i+1], nil
		}
	}

	return "", fmt.Errorf("playlist ID not found in URL")
}

func downloadYtVideoWithRetry(videoURL, outputDir string, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = downloadYtVideo(videoURL, outputDir)
		if err == nil {
			return nil
		}
		fmt.Printf("Attempt %d failed: %v. Retrying...\n", i+1, err)
		time.Sleep(time.Second * 2)
	}
	return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
}

func downloadYtVideo(videoURL, outputDir string) error {
	client := youtube.Client{}
	video, err := client.GetVideo(videoURL)
	if err != nil {
		return fmt.Errorf("failed to get video info: %w", err)
	}

	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return fmt.Errorf("no formats with audio channels available")
	}

	var bestFormat *youtube.Format
	for _, format := range formats {
		if bestFormat == nil || format.AverageBitrate > bestFormat.AverageBitrate {
			bestFormat = &format
		}
	}

	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		return fmt.Errorf("failed to get video stream: %w", err)
	}
	defer stream.Close()

	// Create a temporary file for the audio
	tempFile, err := os.CreateTemp("", "youtube-audio-*.webm")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy the audio stream to the temporary file
	_, err = io.Copy(tempFile, stream)
	if err != nil {
		return fmt.Errorf("failed to save audio: %w", err)
	}

	// Create the output file path
	outputPath := filepath.Join(outputDir, sanitizeFilename(video.Title)+".mp3")

	err = os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert the audio to MP3 using FFmpeg
	buf := bytes.NewBuffer(nil)
	err = ffmpeg_go.Input(tempFile.Name()).
		Output(outputPath, ffmpeg_go.KwArgs{"acodec": "libmp3lame", "b:a": "192k"}).
		WithOutput(buf, buf).
		GlobalArgs("-loglevel", "error", "-hide_banner").
		OverWriteOutput().
		Run()
	if err != nil {
		return fmt.Errorf("failed to convert audio to MP3: %w\nFFmpeg output: %s", err, buf.String())
	}

	fmt.Printf("Downloaded: %s\n", video.Title)
	return nil
}

func sanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	sanitized := strings.Map(func(r rune) rune {
		if r < 32 || r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, filename)

	// Truncate long file names
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}

	return sanitized
}
