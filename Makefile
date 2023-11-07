PLUGIN_NAME="argo-workflows-aws-plugin"
PLUGIN_VERSION:=$(shell cat VERSION | head -1)
GIT_COMMIT:=$(shell git describe --dirty --always)
GIT_BRANCH:=$(shell git rev-parse --abbrev-ref HEAD -- | head -1)
LATEST_GIT_COMMIT:=$(shell git log --format="%H" -n 1 | head -1)
BUILD_USER:=$(shell whoami)
BUILD_DATE:=$(shell date +"%Y-%m-%d")
BUILD_DIR:=$(shell pwd)

all: build_info build
	@echo "$@: complete"

.PHONY: build_info
build_info:
	@echo "Version: $(PLUGIN_VERSION), Branch: $(GIT_BRANCH), Revision: $(GIT_COMMIT)"
	@echo "Build on $(BUILD_DATE) by $(BUILD_USER)"

.PHONY: build
build:
	@echo "$@: started"
	@rm -rf ./bin/$(PLUGIN_NAME)

	@CGO_ENABLED=0 go build -o ./bin/$(PLUGIN_NAME) -v \
		-ldflags="-w -s \
		-X main.appVersion=$(PLUGIN_VERSION) \
		-X main.gitBranch=$(GIT_BRANCH) \
		-X main.gitCommit=$(GIT_COMMIT) \
		-X main.buildUser=$(BUILD_USER) \
		-X main.buildDate=$(BUILD_DATE)" \
		-gcflags="all=-trimpath=$(GOPATH)/src" \
		-asmflags="all=-trimpath $(GOPATH)/src" \
		*.go
	@./bin/$(PLUGIN_NAME) --version
	@echo "$@: complete"

.PHONY: dep
dep:
	@echo "$@: started"
	@versioned || go install github.com/greenpau/versioned/cmd/versioned@latest
	@go install golang.org/x/lint/golint@latest
	@richgo version || go install github.com/kyoh86/richgo@latest
	@echo "$@: complete"

.PHONY: linter
linter:
	@echo "$@: started"
	@golint -set_exit_status *.go
	@for f in `find ./ -type f -name '*.go'`; do echo $$f; go fmt $$f; golint -set_exit_status $$f; done
	@echo "$@: complete"

.PHONY: covdir
covdir:
	@echo "$@: started"
	@mkdir -p .coverage
	@echo "$@: complete"

.PHONY: runtest
runtest:
	@echo "$@: started"
	@go test -v -coverprofile=.coverage/coverage.out ./*.go
	@echo "$@: complete"

.PHONY: test
test: covdir linter runtest coverage
	@echo "$@: complete"

.PHONY: ctest
ctest: covdir linter
	@echo "$@: started"
	@time richgo test -v $(TEST) -coverprofile=.coverage/coverage.out ./*.go
	@echo "$@: complete"

.PHONY: coverage
coverage: covdir
	@echo "$@: started"
	@go tool cover -html=.coverage/coverage.out -o .coverage/coverage.html
	@go test -covermode=count -coverprofile=.coverage/coverage.out ./*.go
	@go tool cover -func=.coverage/coverage.out
	@echo "$@: complete"


.PHONY: qtest
qtest:
	@echo "$@: started"
	@time richgo test -v -run -coverprofile=.coverage/coverage.out -run TestExecutorPlugin *.go
	@echo "$@: complete"

.PHONY: clean
clean:
	@echo "$@: started"
	@rm -rf .doc/ .coverage/ bin/ build/ pkg-build/
	@echo "$@: complete"

.PHONY: release
release:
	@echo "$@: started"
	@go mod tidy
	@go mod verify
	@if [ $(GIT_BRANCH) != "main" ]; then echo "cannot release to non-main branch $(GIT_BRANCH)" && false; fi
	@git diff-index --quiet HEAD -- || ( echo "git directory is dirty, commit changes first" && false )
	@versioned -patch
	@echo "Patched version"
	@git add VERSION
	@versioned -sync main.go
	@git add main.go
	@git commit -m "released v`cat VERSION | head -1`"
	@git tag -a v`cat VERSION | head -1` -m "v`cat VERSION | head -1`"
	@git push
	@git push --tags
	@@echo "If necessary, run the following commands:"
	@echo "  git push --delete origin v$(PLUGIN_VERSION)"
	@echo "  git tag --delete v$(PLUGIN_VERSION)"
	@echo "$@: complete"