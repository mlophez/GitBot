# Makefile

test:
	cd src/; go test

run:
	CONFIG_FILE=config.yaml go run ./cmd/main.go

build-image:
	export DOCKER_BUILDKIT=1
	export BUILDAH_LAYERS=true
	podman build -t docker.io/lgtjpmora/gitbot:dev .

run-image:
	podman run -it --rm --name gitbot --replace docker.io/lgtjpmora/gitbot:dev

publish-image:
	podman push docker.io/lgtjpmora/gitbot:dev


get-token:
	kubectl get secret $(kubectl get serviceaccount my-service-account -o jsonpath='{.secrets[0].name}') -o jsonpath='{.data.token}' | base64 --decode

