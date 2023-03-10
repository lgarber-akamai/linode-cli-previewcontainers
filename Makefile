build-images:
	podman build -t lbgarber/cli-previewcontainers-daemon:latest .
	podman build -t lbgarber/cli-previewcontainers-runner:latest ./runner

push-images: build-images
	podman push lbgarber/cli-previewcontainers-daemon:latest
	podman push lbgarber/cli-previewcontainers-runner:latest