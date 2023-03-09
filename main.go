package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/util/homedir"
	"log"
	"os"
	"path/filepath"
)

func main() {
	kubeConfigDir := homedir.HomeDir()
	if kubeConfigDir != "" {
		kubeConfigDir = filepath.Join(kubeConfigDir, ".kube", "config")
	}

	hostKeyDir := homedir.HomeDir()
	if hostKeyDir != "" {
		hostKeyDir = filepath.Join(hostKeyDir, ".ssh", "id_rsa")
	}

	cliApp := &cli.App{
		Name:  "linode-cli-autodeploy",
		Usage: "Runner daemon for auto-deploying linode-cli testing pods.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "runner-namespace",
				Usage: "The Kubernetes namespace to use for runner pods.",
				Value: "linode-cli-runner",
				EnvVars: []string{"CLI_RUNNER_NAMESPACE"},
			},
			&cli.StringFlag{
				Name:      "kubeconfig",
				Value:     kubeConfigDir,
				EnvVars:   []string{"KUBECONFIG"},
				TakesFile: true,
			},
			&cli.BoolFlag{
				Name: "use-kubeconfig",
			},
			&cli.IntFlag{
				Name:  "max-concurrent-runners",
				Usage: "The maxiumum number of runners a user can have at a time.",
				Value: 3,
			},
			&cli.StringFlag{
				Name: "ssh-hostkey",
				Usage: "The host key to use for the simulated SSH server.",
				Value: hostKeyDir,
				EnvVars: []string{"CLI_HOST_KEY"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Serve the SSH server to incoming connections",
				Action: func(context *cli.Context) error {
					if err := serve(context); err != nil {
						return fmt.Errorf("failed to serve: %s", err)
					}

					return nil
				},
			},
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
