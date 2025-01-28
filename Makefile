git_commit = $(shell git log -1 --pretty=format:"%H")

test_folder ?= container

test_unit:
	dagger call --mod=dagger --progress=plain unit-test \
		--src=. \
		--with-race=false

test_functional_all:
	dagger call --mod=dagger functional-test-all \
		--src=$(shell pwd)/examples \
		--jumppad=$(which jumppad)

test_functional_podman:
	GOOS=linux go build -ldflags "-X main.version=${git_commit}" -gcflags=all="-N -l" -o bin/jumppad main.go
	dagger call --mod=dagger functional-test \
		--src=$(shell pwd)/examples \
		--working-directory=$(test_folder) \
		--runtime=podman \
		--jumppad=./bin/jumppad

test_functional_docker:
	GOOS=linux go build -ldflags "-X main.version=${git_commit}" -gcflags=all="-N -l" -o bin/jumppad main.go
	dagger call --mod=dagger functional-test \
		--src=$(shell pwd)/examples \
		--working-directory=$(test_folder) \
		--runtime=docker \
		--jumppad=./bin/jumppad

test_e2e_cmd: install_local
	jumppad up --no-browser ./examples/single_k3s_cluster
	jumppad down

dagger_build:
	dagger call -m dagger all \
		--output=./all_archive \
		--src=. \
		--notorize-cert=${QUILL_SIGN_P12} \
		--notorize-cert-password=QUILL_SIGN_PASSWORD \
		--notorize-key=${QUILL_NOTARY_KEY} \
		--notorize-id=${QUILL_NOTARY_KEY_ID} \
		--notorize-issuer=${QUILL_NOTARY_ISSUER}

dagger_release:
	dagger call -m dagger release \
		--github-token=GITHUB_TOKEN \
		--archives=./all_archive \
		--src=.  

generate_mocks:
	go install github.com/vektra/mockery/v2@v2.46.0
	go generate ./...

install_local:
	go build -ldflags "-X main.version=${git_commit}" -gcflags=all="-N -l" -o bin/jumppad main.go
	sudo mv /usr/bin/jumppad /usr/local/bin/jumppad-old || true
	sudo cp bin/jumppad /usr/local/bin/jumppad

remove_local:
	sudo mv /usr/local/bin/jumppad-old /usr/local/bin/jumppad || true