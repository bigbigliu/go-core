GO ?= go
PROJECT ?= go-core
IMAGE ?= $(PROJECT)
VERSION ?= latest

.PHONY: lint fmt test vet race tidy compile build docker-push git-commit git-push pre-push

lint:
	golangci-lint run --issues-exit-code 1

fmt:
	@gofmt -w .
	@goimports -w .

test:
	@$(GO) test ./... -v

vet:
	@$(GO) vet ./...

race:
	@$(GO) test -race ./...

tidy:
	@$(GO) mod tidy

compile:
	@mkdir -p target
	@GOOS=linux GOARCH=amd64 $(GO) build -trimpath -o target/$(PROJECT)-linux-amd64 ./examples/basic
	@GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -o target/$(PROJECT)-darwin-amd64 ./examples/basic
	@GOOS=windows GOARCH=amd64 $(GO) build -trimpath -o target/$(PROJECT)-windows-amd64.exe ./examples/basic

build:
	@docker build -t $(IMAGE):$(VERSION) .
	@echo "build successful"

docker-push:
	docker push $(IMAGE):$(VERSION)

pre-push:
	@commit_message=`git log -1 --pretty=format:"%s"`; \
	if [ -z "$$commit_message" ]; then \
		echo "Error: No commit message provided. Please use 'git commit -m' to add a commit message."; \
		exit 1; \
	fi

git-commit:
	@git add .
	@git commit -m "$(COMMIT)"

git-push: pre-push
	@git push origin $(BRANCH)
