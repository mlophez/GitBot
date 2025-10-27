# Makefile

test:
	cd src/; go test

run:
	CONFIG_FILE=config.yaml go run ./cmd/main.go

build-image:
	export DOCKER_BUILDKIT=1
	export BUILDAH_LAYERS=true
	podman build --platform linux/amd64 -f Containerfile -t 234166862235.dkr.ecr.eu-south-2.amazonaws.com/gitops-bot:dev .

run-image:
	podman run -it --rm --name gitbot --replace 234166862235.dkr.ecr.eu-south-2.amazonaws.com/gitops-bot:dev

publish-image:
	podman push 234166862235.dkr.ecr.eu-south-2.amazonaws.com/gitops-bot:dev


get-token:
	kubectl get secret $(kubectl get serviceaccount my-service-account -o jsonpath='{.secrets[0].name}') -o jsonpath='{.data.token}' | base64 --decode

