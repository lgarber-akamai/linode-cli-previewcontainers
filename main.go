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
	home := homedir.HomeDir()
	if home != "" {
		home = filepath.Join(home, ".kube", "config")
	}

	cliApp := &cli.App{
		Name:  "linode-cli-autodeploy",
		Usage: "Runner daemon for auto-deploying linode-cli testing pods.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "runner-namespace",
				Usage: "The Kubernetes namespace to use for runner pods.",
				Value: "linode-cli-runner",
			},
			&cli.StringFlag{
				Name:      "kubeconfig",
				Value:     home,
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
