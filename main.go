package main

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/lyon-v/squash-docker-image/internal/image"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Version of the application, should be set during build
var Version = "1.0.0"

var (
	verbose    bool
	version    bool
	imageName  string
	fromLayer  string
	tag        string
	message    string
	cleanup    bool
	tmpDir     string
	outputPath string
	loadImage  bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "squash-docker-image",
		Short: "squash-docker-image is a CLI for squashing Docker images",
		Run: func(cmd *cobra.Command, args []string) {
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
			if version {
				fmt.Println("Version:", Version)
				return
			}

			// Validate required flags
			if imageName == "" {
				logger.Error("Image is required")
				cmd.Usage()
				return
			}

			// Set log level
			if verbose {
				logger.SetLevel(logrus.DebugLevel)
				logger.Debug("Verbose mode enabled")
			} else {
				logger.SetLevel(logrus.InfoLevel)
			}

			logger.Infof("Running version %s", Version)

			// Create Squash instance
			cli := image.CLI{
				Verbose:    verbose,
				Version:    version,
				Image:      imageName,
				FromLayer:  fromLayer,
				Tag:        tag,
				Message:    message,
				Cleanup:    cleanup,
				TmpDir:     tmpDir,
				OutputPath: outputPath,
				LoadImage:  loadImage,
			}

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
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&version, "version", "V", false, "Show version and exit")
	rootCmd.Flags().StringVarP(&imageName, "image", "i", "", "Image to be squashed (required)")
	rootCmd.Flags().StringVarP(&fromLayer, "from-layer", "f", "", "Number of layers to squash or ID of the layer to squash from")
	rootCmd.Flags().StringVarP(&tag, "tag", "t", "", "Specify the tag to be used for the new image")
	rootCmd.Flags().StringVarP(&message, "message", "m", "squash image", "Specify a commit message for the new image")
	rootCmd.Flags().BoolVarP(&cleanup, "cleanup", "c", false, "Remove source image from Docker after squashing")
	rootCmd.Flags().StringVarP(&tmpDir, "tmp-dir", "d", "", "Temporary directory to be created and used")
	rootCmd.Flags().StringVarP(&outputPath, "output-path", "o", "", "Path where the image may be stored after squashing")
	rootCmd.Flags().BoolVarP(&loadImage, "load-image", "l", true, "Whether to load the image into Docker daemon after squashing")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
