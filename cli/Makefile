MESSAGE = github.com/metaflowys/metaflow/message
LIBS = github.com/metaflowys/metaflow/server/libs
SERVER = github.com/metaflowys/metaflow/server
REV_COUNT = $(shell git rev-list --count HEAD)
COMMIT_DATE = $(shell git show -s --format=%cd --date=short HEAD)
REVISION = $(shell git rev-parse HEAD)
FLAGS = -gcflags "-l -l" -ldflags "-X main.RevCount=${REV_COUNT} -X main.Revision=${REVISION} -X main.CommitDate=${COMMIT_DATE} \
		-X 'main.goVersion=$(shell go version)'"
BINARY_SUFFIX :=

.PHONY: all
all: cli

vendor:
	go mod tidy && go mod download && go mod vendor
	cp -r ../message/* vendor/${MESSAGE}/
	cp -r ../server/go.* vendor/${SERVER}/
	find vendor -type d -exec chmod +w {} \;
	cp vendor/${MESSAGE}/metric.proto vendor/${LIBS}/zerodoc/pb
	cp vendor/${MESSAGE}/flow_log.proto vendor/${LIBS}/datatype/pb
	sed -i '6,10s/^...//' vendor/${LIBS}/zerodoc/pb/metric.proto
	sed -i '6,10s/^...//' vendor/${LIBS}/datatype/pb/flow_log.proto
	cd vendor/${MESSAGE} && go generate ./...
	cd vendor/${LIBS} && go generate ./...

.PHONY: cli
cli: vendor
	go build -mod vendor ${FLAGS} -o bin/metaflow-ctl${BINARY_SUFFIX} cmd/ctl/main.go

.PHONY: clean
clean:
	touch vendor
	chmod -R 777 vendor
	rm -rf vendor
	rm -rf bin
