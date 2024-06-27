package image

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

const DefaultTimeoutSeconds = 600

func NewDockerClient(logger *logrus.Logger) (*client.Client, error) {
	// Read and parse timeout from environment variable
	timeoutSeconds := readEnvOrDefault("DOCKER_TIMEOUT", DefaultTimeoutSeconds)
	if timeoutSeconds <= 0 {
		return nil, logError("Provided timeout value needs to be greater than zero.")
	}

	// Set Docker host from environment variable if deprecated DOCKER_CONNECTION is used
	if dockerConnection := os.Getenv("DOCKER_CONNECTION"); dockerConnection != "" {
		os.Setenv("DOCKER_HOST", dockerConnection)
		logger.Println("DOCKER_CONNECTION is deprecated, please use DOCKER_HOST instead")
	}

	// Create a new Docker client with auto-detected API version and timeout
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithTimeout(time.Duration(timeoutSeconds)*time.Second))
	if err != nil {
		return nil, logError("Could not create Docker client: " + err.Error())
	}

	return cli, nil
}

func readEnvOrDefault(envKey string, defaultValue int) int {
	value, exists := os.LookupEnv(envKey)
	if !exists {
		return defaultValue
	}
	timeout, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("Provided timeout value: %s cannot be parsed as integer, exiting.", value)
	}
	return timeout
}

func validDockerConnection(cli *client.Client) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := cli.Ping(ctx)
	return err == nil
}

func logError(msg string) error {
	log.Println(msg)
	return logError(msg)
}
