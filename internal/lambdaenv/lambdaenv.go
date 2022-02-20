package lambdaenv

import (
	"context"
	"fmt"
	"sync"

	"github.com/mindriot101/lambda-local-runner/internal/docker"
	"github.com/rs/zerolog/log"
)

/* lambdaenv abstracts the docker commands into a lambda specifc API
*
*
 */

// dockerAPI is our consumed API surface from the docker package
type dockerAPI interface {
	Run(context.Context, *docker.RunArgs) error
}

type LambdaEnvironment struct {
	api dockerAPI

	mu        sync.Mutex
	usedPorts []int
	lastPort  int
}

func New(api dockerAPI) *LambdaEnvironment {
	return &LambdaEnvironment{
		api:      api,
		lastPort: 9000,
	}
}

// Spawn is designed to run in a background goroutine
//
// Spawn runs a lambda function in a container and exposes the container port
// on a unique port in the system. The port is then used to invoke the lambda
// with the incoming payload from the HTTP request.
func (e *LambdaEnvironment) Spawn(ctx context.Context, runtime string, handler string) error {
	// docker run \
	// 	--rm \
	// 	-it \
	// 	-e DOCKER_LAMBDA_STAY_OPEN=1 \
	// 	-p 9001:9001 \
	// 	-v /home/simon/dev/lambda-local-runner/testproject/.aws-sam/build/HelloWorldFunction:/var/task:ro,delegated \
	// 	lambci/lambda:python3.8 \
	// 	app.lambda_handler \
	// 	'{}'
	//
	port := e.newPort()
	imageName := fmt.Sprintf("lambci/lambda:%s", runtime)
	args := &docker.RunArgs{
		Image: imageName,
		// FIXME
		MountDir:    "/home/simon/dev/lambda-local-runner/testproject/.aws-sam/build/HelloWorldFunction",
		ExposedPort: port,
		Command:     []string{handler},
	}
	log.Debug().Interface("args", args).Msg("running container")
	if err := e.api.Run(ctx, args); err != nil {
		return fmt.Errorf("error running lambda container: %w", err)
	}
	return nil
}

func (e *LambdaEnvironment) newPort() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	port := e.lastPort + 1
	e.lastPort = port
	e.usedPorts = append(e.usedPorts, port)

	return port
}
