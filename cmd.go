package dockerize_proxy

import (
	"log"

	"context"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func init() {
	caddy.RegisterModule(MyPlugin{})
}

type MyPlugin struct {
}

func (MyPlugin) CaddyModule() caddy.ModuleInfo {
	go func() {
		err := helloDocker()
		if err != nil {
			panic(err)
		}
	}()

	return caddy.ModuleInfo{
		ID:  "hello.world",
		New: func() caddy.Module { return new(MyPlugin) },
	}
}

func helloDocker() error {
	// ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	dockerClient, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	dockerPing, err := dockerClient.Ping(context.Background())
	if err != nil {
		return err
	}

	dockerClient.NegotiateAPIVersionPing(dockerPing)

	args := filters.NewArgs()
	args.Add("type", "service")
	args.Add("type", "container")
	args.Add("type", "config")

	context, cancel := context.WithCancel(context.Background())

	eventsChan, errorChan := dockerClient.Events(context, types.EventsOptions{
		Filters: args,
	})

	log.Println("Connecting to docker events")

ListenEvents:
	for {
		select {
		case event := <-eventsChan:
			update := (event.Type == "container" && event.Action == "create") ||
				(event.Type == "container" && event.Action == "start") ||
				(event.Type == "container" && event.Action == "stop") ||
				(event.Type == "container" && event.Action == "die") ||
				(event.Type == "container" && event.Action == "destroy") ||
				(event.Type == "service" && event.Action == "create") ||
				(event.Type == "service" && event.Action == "update") ||
				(event.Type == "service" && event.Action == "remove") ||
				(event.Type == "config" && event.Action == "create") ||
				(event.Type == "config" && event.Action == "remove")

			if update {
				log.Println("Docker event", event)
			}
		case err := <-errorChan:
			cancel()
			if err != nil {
				log.Println("Docker events error", zap.Error(err))
			}
			break ListenEvents
		}
	}

	return nil
}
