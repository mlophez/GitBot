# Makefile

test:
	cd src/; go test

run:
	go run ./cmd/main.go

build-image:
	podman build -t docker.io/lgtjpmora/gitbot:dev .

run-image:
	podman run -it --rm --name gitbot --replace docker.io/lgtjpmora/gitbot:dev

publish-image:
	podman push docker.io/lgtjpmora/gitbot:dev


