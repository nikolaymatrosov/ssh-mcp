package testcontainers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SSHContainer represents an SSH server container
type SSHContainer struct {
	Container testcontainers.Container
	Host      string
	Port      int
}

// StartSSHContainer starts an SSH server container
func StartSSHContainer(ctx context.Context) (*SSHContainer, error) {
	// Get the absolute path to the Dockerfile
	dockerfilePath, err := filepath.Abs("../e2e/testcontainers/Dockerfile")
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to Dockerfile: %v", err)
	}

	// Create a request for a container with the SSH server
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    filepath.Dir(dockerfilePath),
			Dockerfile: "Dockerfile",
		},
		ExposedPorts: []string{"22/tcp"},
		WaitingFor:   wait.ForListeningPort("22/tcp").WithStartupTimeout(30 * time.Second),
	}

	// Start the container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %v", err)
	}

	// Get the container's host and port
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "22/tcp")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped port: %v", err)
	}

	return &SSHContainer{
		Container: container,
		Host:      host,
		Port:      mappedPort.Int(),
	}, nil
}

// Stop stops the SSH server container
func (c *SSHContainer) Stop(ctx context.Context) error {
	return c.Container.Terminate(ctx)
}
