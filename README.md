# YouTube Playlist Downloader

A concurrent Go application that downloads and converts YouTube playlist videos to MP3 format.

## Features

- Downloads entire YouTube playlists
- Converts videos to high-quality MP3 files (192kbps)
- Concurrent downloading for improved speed
- User-friendly GUI directory selector
- Automatic retry mechanism for failed downloads
- Error handling and reporting
- Sanitized filenames for cross-platform compatibility

## Requirements

- Go 1.16 or higher
- FFmpeg installed on your system. If not installed, download it from [here](https://www.ffmpeg.org/download.html)
- Internet connection

## Installation

1. Clone the repository:
    ```bash
    git clone https://github.com/ndzuma/PlaylistDownloader.git
    cd PlaylistDownloader
    ```

2. Install dependencies:
    ```bash
    go mod download
    ```

3. Build the application:
    ```bash
    go build
    ```

## Usage

1. Run the application
2. Enter a public YouTube playlist URL when prompted
3. Select the output directory using the file dialog
4. Choose a custom folder name or use the default playlist name
5. Wait for the downloads to complete

## Technical Details

### Key Components

- **Concurrent Downloads**: Uses Go routines with a semaphore to limit concurrent downloads
- **Error Handling**: Implements retry mechanism for failed downloads
- **Format Selection**: Automatically selects the highest quality audio stream
- **File Management**: Creates temporary files for processing and cleans up afterward

### Dependencies

- github.com/kkdai/youtube/v2: YouTube video downloading
- github.com/sqweek/dialog: GUI directory selection
- github.com/u2takey/ffmpeg-go: FFmpeg wrapper for audio conversion
- Standard Go libraries for file operations and concurrency

## Potential Improvements

### Functionality
- Add support for spotify, soundcloud, and other music platforms playlists
- Add progress bars for individual downloads
- Add batch playlist processing
- Implement download queue management

### Performance
- Optimize concurrent download limits based on system resources
- Implement smart retry mechanisms with exponential backoff

### User Experience
- Create a full GUI interface
- Include audio quality selection options

### Technical
- Bundle FFmpeg with the application
- Implement proper logging system
- Add unit tests and integration tests
- Implement proper error recovery mechanisms
- Add configuration file support

### Security
- Implement rate limiting
- Add checksum verification for downloads
- Implement secure temporary file handling


## Contributing

This project is open for contributions. Feel free to:
- Fork the repository
- Create a feature branch
- Commit your changes
- Push to the branch
- Open a Pull Request

Note: By contributing to this project, you agree that your contributions will be licensed under its MIT License.
This software is provided as-is, and while contributions are welcome, the maintainers are not responsible for any issues that may arise from its use.
