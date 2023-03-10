# linode-cli-previewcontainers

An unfinished experimental project for running and managing short-lived "preview" containers in Kubernetes.

This project was written for fun as a proof-of-concept, so it definitely needs some refactoring (and a kustomization.yaml file).

## Daemon Usage

```
linode-cli-autodeploy [global options] command [command options] [arguments...]

COMMANDS:
   serve    Serve the SSH server to incoming connections
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --runner-namespace value        The Kubernetes namespace to use for runner pods. (default: "linode-cli-runner") [$CLI_RUNNER_NAMESPACE]
   --runner-image value            The image to deploy on runner pods. (default: "lbgarber/cli-previewcontainers-runner:latest")
   --kubeconfig value              (default: "/Users/myuser/.kube/config") [$KUBECONFIG]
   --use-kubeconfig                (default: false)
   --max-concurrent-runners value  The maxiumum number of runners a user can have at a time. (default: 3)
   --ssh-hostkey value             The host key to use for the simulated SSH server. (default: "/Users/lgarber/.ssh/id_rsa") [$CLI_HOST_KEY]
   --runner-memory-limit value     The memory limit for runner containers. (default: "512M")
   --runner-cpu-limit value        The CPU limit for runner containers. (default: "0.6")
   --ssh-listen-port value         The port for the simulated SSH server to listen on (default: 2222)
   --runner-expiry-time value      The time (in minutes) until a runner session should automatically be destroyed (default: 15)
   --help, -h                      show help
```
