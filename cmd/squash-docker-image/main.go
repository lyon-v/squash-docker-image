package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"squash-docker-image/internal/image"
	"strings"

	"github.com/sirupsen/logrus"
)

// Version of the application, should be set during build
var Version string = "1.0.0"

// CLI holds the command line arguments

func run() {
	cli := image.CLI{}
	flag.BoolVar(&cli.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&cli.Version, "version", false, "Show version and exit")
	flag.StringVar(&cli.Image, "image", "", "Image to be squashed (required)")
	flag.StringVar(&cli.FromLayer, "from-layer", "", "Number of layers to squash or ID of the layer to squash from")
	flag.StringVar(&cli.Tag, "tag", "", "Specify the tag to be used for the new image")
	flag.BoolVar(&cli.Force, "force", false, "Force squash image if not match option")
	flag.StringVar(&cli.Message, "message", "squash image", "Specify a commit message for the new image")
	flag.BoolVar(&cli.Cleanup, "cleanup", false, "Remove source image from Docker after squashing")
	flag.StringVar(&cli.TmpDir, "tmp-dir", "", "Temporary directory to be created and used")
	flag.StringVar(&cli.OutputPath, "output-path", "", "Path where the image may be stored after squashing")
	flag.BoolVar(&cli.LoadImage, "load-image", true, "Whether to load the image into Docker daemon after squashing")

	flag.Parse()

	loggers := logrus.New()
	loggers.SetLevel(logrus.InfoLevel)
	loggers.SetReportCaller(true)
	loggers.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "2006-01-02 15:03:04",
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := fmt.Sprintf("%s, line:%d", path.Base(frame.File), frame.Line)
			funcs := strings.Split(frame.Function, ".")
			strfuncs := funcs[len(funcs)-1]
			return strfuncs, fileName
		},
	})

	if cli.Version {
		loggers.Info(Version)
		os.Exit(0)
	}

	if cli.Image == "" {
		loggers.Info("Error: Image is required")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if cli.Verbose {
		loggers.SetLevel(logrus.DebugLevel)
		loggers.Info("Verbose mode enabled")
	}

	loggers.Infof("Running version %s", Version)

	// Here you would integrate the actual squashing logic, possibly
	// using a function or a package dedicated to squashing Docker images
	// e.g., squash.SquashImage(cli)

	squash, err := image.NewSquash(cli, loggers)
	if err != nil {
		loggers.Fatal("error: ", err)
	}
	newImageId, err := squash.Run()
	if err != nil {
		loggers.Fatal("error: ", err)
	}
	fmt.Printf("suqashed imageId: [%s] \n", newImageId)

}

func main() {
	run()
}
