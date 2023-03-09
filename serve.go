package main

import (
	"fmt"
	"github.com/gliderlabs/ssh"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
	"linode-cli-autodeploy/appcontext"
	"log"
	"strconv"
	"strings"
)

func getSSHHandler(appContext *appcontext.AppContext) ssh.Handler {
	return func(s ssh.Session) {
		remoteAddressSegments := strings.Split(s.RemoteAddr().String(), ":")
		remoteAddress := strings.Join(remoteAddressSegments[:len(remoteAddressSegments)-1], ":")

		term := terminal.NewTerminal(s, "> ")

		term.Write([]byte("GitHub PR #:\n"))

		prNumberStr, err := term.ReadLine()
		if err != nil {
			log.Printf("[WARN] failed to read pr number: %s\n", err)
			return
		}

		prNumber, err := strconv.Atoi(prNumberStr)
		if err != nil {
			log.Printf("[WARN] failed to parse pr number: %s\n", err)
			term.Write([]byte(fmt.Sprintf("ERROR: Non-numeric PR number: %s\n", prNumberStr)))
			return
		}

		pr, err := fetchPRDetails(prNumber)
		if err != nil {
			log.Printf("[WARN] failed to fetch PR details for PR #%d: %s\n", prNumber, err)
			return
		}

		term.Write([]byte("Linode API Token:\n"))

		token, err := term.ReadLine()
		if err != nil {
			log.Printf("[WARN] failed to read linode token: %s\n", err)
			return
		}

		canProvision, err := appContext.CanOriginProvisionRunner(remoteAddress)
		if err != nil {
			log.Printf("[WARN] Failed to check pods for user rate-limit: %s", err)
			return
		}

		if !canProvision {
			s.Write([]byte(
				fmt.Sprintf("You have exceeded the rate limit of %d concurrent sessions. "+
					"Please wait for existing sessions to clean up.\n", appContext.MaxConcurrentRunners)))
			return
		}

		term.Write([]byte("Provisioning CLI environment...\n"))

		// Create a runner instance
		runner, err := appContext.ProvisionRunner(appcontext.ProvisionRunnerOptions{
			RepoCloneURL: *pr.Head.Repo.CloneURL,
			RepoBranch:   *pr.Head.Ref,
			Origin:       remoteAddress,
			Token:        token,
		})
		if err != nil {
			log.Printf("[WARN] failed to provision runner: %s\n", err)
			term.Write([]byte("Failed to provision CLI environment :(\n"))
			return
		}

		defer func() {
			if err := appContext.DestroyRunner(runner); err != nil {
				log.Printf("[WARN] failed to clean up runner: %s", err)
			}
		}()

		term.Write([]byte("Attaching to environment...\n"))

		// Attach the SSH session to the runner
		if err := appContext.AttachRunner(runner, s); err != nil {
			log.Printf("[WARN] failed to attach runner: %s\n", err)
			term.Write([]byte("Failed to attach to CLI environment :("))
			return
		}
	}
}

func serve(context *cli.Context) error {
	// Create app context

	log.Println("[INFO] Creating app context")

	appContext, err := appcontext.NewAppContext(appcontext.AppContextOptions{
		KubeConfig:           context.String("kubeconfig"),
		UseKubeConfig:        context.Bool("use-kubeconfig"),
		Namespace:            context.String("runner-namespace"),
		MaxConcurrentRunners: context.Int("max-concurrent-runners"),
	})
	if err != nil {
		return fmt.Errorf("failed to create app context: %s", err)
	}

	log.Println("[INFO] Initializing app context")
	if err := appContext.Init(); err != nil {
		return fmt.Errorf("failed to initialize app: %s", err)
	}

	go appContext.RunnerCleanupCron()

	log.Printf("[INFO] Listening on port 2222")

	return ssh.ListenAndServe(
		":2222",
		getSSHHandler(appContext),
		ssh.HostKeyFile(context.String("ssh-hostkey")))
}
