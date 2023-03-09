build-images:
	podman build -t lbgarber/cli-autodeploy-daemon:latest .
	podman build -t lbgarber/cli-autodeploy-runner:latest ./runner

push-images: build-images
	podman push lbgarber/cli-autodeploy-daemon:latest
	podman push lbgarber/cli-autodeploy-runner:latest