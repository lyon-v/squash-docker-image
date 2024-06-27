package image

import (
	"errors"
	"fmt"

	// "log"
	"os"

	"github.com/docker/docker/client"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type CLI struct {
	Verbose    bool
	Version    bool
	Image      string
	FromLayer  string
	Tag        string
	Force      bool
	Message    string
	Cleanup    bool
	TmpDir     string
	OutputPath string
	LoadImage  bool
}

// Squash represents the main structure to handle Docker image squashing.
type Squash struct {
	// log         *log.Logger
	logs          *logrus.Logger
	docker        *client.Client
	image         string
	fromLayer     string
	tag           string
	force         bool
	comment       string
	tmpDir        string
	outputPath    string
	loadImage     bool
	cleanup       bool
	development   bool
	lastCreatedBy string
}

// NewSquash creates a new Squash instance.
func NewSquash(cli CLI, loggers *logrus.Logger) (*Squash, error) {

	if loggers == nil {
		loggers := logrus.New()
		// 可以从环境变量读取或者直接指定日志级别
		if os.Getenv("DEBUG") == "true" {
			loggers.SetLevel(logrus.DebugLevel)
		} else {
			loggers.SetLevel(logrus.InfoLevel)
		}
	}

	var dockerClient *client.Client
	var err error
	if dockerClient, err = NewDockerClient(loggers); err != nil {
		return nil, err
	}
	development := false

	if len(cli.TmpDir) != 0 {
		development = true
	}
	if cli.Tag == cli.Image && cli.Cleanup {
		cli.Cleanup = false
	}

	return &Squash{
		logs:        loggers,
		docker:      dockerClient,
		image:       cli.Image,
		fromLayer:   cli.FromLayer,
		tag:         cli.Tag,
		force:       cli.Force,
		comment:     cli.Message,
		tmpDir:      cli.TmpDir,
		outputPath:  cli.OutputPath,
		loadImage:   cli.LoadImage,
		cleanup:     cli.Cleanup,
		development: development,
	}, nil
}

// Run executes the squashing process.
func (s *Squash) Run() (string, error) {

	ctx := context.Background()
	dockerVersion, err := s.docker.ServerVersion(ctx)
	if err != nil {
		s.logs.Error("Could not get the version of dockerserver %s: %v\n", s.docker, err)
		return "", err
	}

	s.logs.Infof("docker-squash version %s, Docker %s, API %s...", squashVersion, dockerVersion.Version, dockerVersion.APIVersion)

	if len(s.image) == 0 {
		return "", errors.New("image is not provided")
	}

	if len(s.outputPath) == 0 && !s.loadImage {
		// log.Println("No output path specified and loading into Docker is not selected either; squashed image would not be accessible, proceeding with squashing doesn't make sense")
		return "", fmt.Errorf("No output path specified and loading into Docker is not selected either; squashed image would not be accessible, proceeding with squashing doesn't make sense")
	}

	// Check if the output path already exists
	if _, err := os.Stat(s.outputPath); err == nil {
		s.logs.Infof("Path '%s' specified as output path where the squashed image should be saved already exists, it'll be overridden", s.outputPath)
	} else if !os.IsNotExist(err) {
		s.logs.Fatalf("Failed to check if output path exists: %v", err)
	}

	minVersion, _ := version.NewVersion("1.22")
	dockerAPIVersion, _ := version.NewVersion(dockerVersion.APIVersion)

	var img ImageInterface
	//留着拓展
	if dockerAPIVersion.GreaterThanOrEqual(minVersion) {
		img = NewV2Image(s)
	}
	s.logs.Println("Squashing image:", s.image)
	if s.outputPath != "" {
		// Simulate exporting tar archive
		s.logs.Infof("Exporting squashed image to %s\n", s.outputPath)
	}

	var newImageId string
	if err, newImageId = s.squash(img); err != nil {
		return "", nil
	}

	s.logs.Println("Squashing complete")

	return newImageId, nil
}

func (s *Squash) squash(img ImageInterface) (error, string) {

	newImageId, err := img.Squash()

	if err != nil {
		s.logs.Errorf("error squashing the image %s", err.Error())

		return err, ""
	}

	if len(s.outputPath) != 0 {
		img.ExportTarArchive(s.outputPath)
	}
	if s.loadImage {
		if err := img.LoadSquashedImage(); err != nil {
			return err, ""
		}
	}

	if s.cleanup {
		img.Cleanup()
		s.logs.Info("Cleaning up source image")
	}

	return nil, newImageId
}
