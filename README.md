# linode-cli-previewcontainers

An unfinished experimental project for running and managing short-lived "preview" containers in Kubernetes.

This project was written for fun as a proof-of-concept, so it definitely needs some refactoring.

## Daemon Usage

```
NAME:
   linode-cli-autodeploy - Runner daemon for auto-deploying linode-cli testing pods.

USAGE:
   linode-cli-autodeploy [global options] command [command options] [arguments...]

COMMANDS:
   serve    Serve the SSH server to incoming connections
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --runner-namespace value        The Kubernetes namespace to use for runner pods. (default: "linode-cli-runner") [$CLI_RUNNER_NAMESPACE]
   --runner-image value            The image to deploy on runner pods. (default: "lbgarber/cli-previewcontainers-runner:latest") [$CLI_RUNNER_IMAGE]
   --kubeconfig value              (default: "/Users/lgarber/.kube/config") [$KUBECONFIG]
   --use-kubeconfig                (default: false)
   --max-concurrent-runners value  The maxiumum number of runners a user can have at a time. (default: 3) [$CLI_MAX_CONCURRENT_RUNNERS]
   --ssh-hostkey value             The host key to use for the simulated SSH server. (default: "/Users/lgarber/.ssh/id_rsa") [$CLI_HOST_KEY]
   --runner-memory-limit value     The memory limit for runner containers. (default: "512M") [$CLI_RUNNER_MEMORY_LIMIT]
   --runner-cpu-limit value        The CPU limit for runner containers. (default: "0.6") [$CLI_RUNNER_CPU_LIMIT]
   --ssh-listen-port value         The port for the simulated SSH server to listen on (default: 2222) [$CLI_SSH_PORT]
   --runner-expiry-time value      The time (in minutes) until a runner session should automatically be destroyed (default: 15) [$CLI_RUNNER_EXPIRY_TIME]
   --help, -h                      show help
```
