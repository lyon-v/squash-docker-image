package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/lyon-v/squash-docker-image/internal/image"

	"github.com/sirupsen/logrus"
)

// Version of the application, should be set during build
var Version = "1.0.0"

func main() {
	// Parse CLI arguments
	cli := image.CLI{}
	flag.BoolVar(&cli.Verbose, "verbose", false, "Verbose output (shorthand: -v)")
	flag.BoolVar(&cli.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&cli.Version, "version", false, "Show version and exit (shorthand: -V)")
	flag.BoolVar(&cli.Version, "V", false, "Show version and exit")
	flag.StringVar(&cli.Image, "image", "", "Image to be squashed (required) (shorthand: -i)")
	flag.StringVar(&cli.Image, "i", "", "Image to be squashed (required)")
	flag.StringVar(&cli.FromLayer, "from-layer", "", "Number of layers to squash or ID of the layer to squash from (shorthand: -f)")
	flag.StringVar(&cli.FromLayer, "f", "", "Number of layers to squash or ID of the layer to squash from")
	flag.StringVar(&cli.Tag, "tag", "", "Specify the tag to be used for the new image (shorthand: -t)")
	flag.StringVar(&cli.Tag, "t", "", "Specify the tag to be used for the new image")
	flag.StringVar(&cli.Message, "message", "squash image", "Specify a commit message for the new image (shorthand: -m)")
	flag.StringVar(&cli.Message, "m", "squash image", "Specify a commit message for the new image")
	flag.BoolVar(&cli.Cleanup, "cleanup", false, "Remove source image from Docker after squashing (shorthand: -c)")
	flag.BoolVar(&cli.Cleanup, "c", false, "Remove source image from Docker after squashing")
	flag.StringVar(&cli.TmpDir, "tmp-dir", "", "Temporary directory to be created and used (shorthand: -d)")
	flag.StringVar(&cli.TmpDir, "d", "", "Temporary directory to be created and used")
	flag.StringVar(&cli.OutputPath, "output-path", "", "Path where the image may be stored after squashing (shorthand: -o)")
	flag.StringVar(&cli.OutputPath, "o", "", "Path where the image may be stored after squashing")
	flag.BoolVar(&cli.LoadImage, "load-image", true, "Whether to load the image into Docker daemon after squashing (shorthand: -l)")
	flag.BoolVar(&cli.LoadImage, "l", true, "Whether to load the image into Docker daemon after squashing")
	flag.Parse()

	// Initialize logger
	logger := logrus.New()
	logger.SetReportCaller(true)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "2006-01-02 15:03:04",
		CallerPrettyfier: func(frame *runtime.Frame) (string, string) {
			return fmt.Sprintf("%s", strings.Split(frame.Function, ".")[len(strings.Split(frame.Function, "."))-1]),
				fmt.Sprintf("%s, line:%d", path.Base(frame.File), frame.Line)
		},
	})

	// Handle version flag
	if cli.Version {
		fmt.Println("Version:", Version)
		return
	}

	// Validate required flags
	if cli.Image == "" {
		logger.Error("Image is required")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Set log level
	if cli.Verbose {
		logger.SetLevel(logrus.DebugLevel)
		logger.Debug("Verbose mode enabled")
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.Infof("Running version %s", Version)

	// Create Squash instance
	squash, err := image.NewSquash(cli, logger)
	if err != nil {
		logger.Fatalf("Failed to create Squash instance: %v", err)
	}

	// Run squash process
	newImageId, err := squash.Run()
	if err != nil {
		logger.Fatalf("Squash process failed: %v", err)
	}

	fmt.Printf("Squashed image ID: [%s]\n", newImageId)
}
