MESSAGE = github.com/metaflowys/metaflow/message
REV_COUNT = $(shell git rev-list --count HEAD)
COMMIT_DATE = $(shell git show -s --format=%cd --date=short HEAD)
REVISION = $(shell git rev-parse HEAD)
FLAGS = -gcflags "-l -l" -ldflags "-X main.RevCount=${REV_COUNT} -X main.Revision=${REVISION} -X main.CommitDate=${COMMIT_DATE} \
		-X 'main.goVersion=$(shell go version)'"
BINARY_SUFFIX :=

.PHONY: all
all: server

generate_sources = $(shell find -not -path './vendor/*' -name '*.go.tmpl' -or -name '*.tmpldata')
generated_libs = libs/hmap/idmap/ubig_id_map.go libs/hmap/idmap/ubig_id_map_test.go \
		 libs/hmap/lru/ubig_lru.go libs/hmap/lru/ubig_lru_test.go \
		 libs/zerodoc/pooled_meters.go

proto = vendor/${MESSAGE}/common/common.pb.go vendor/${MESSAGE}/trident/trident.pb.go libs/datatype/pb/flow_log.pb.go libs/zerodoc/pb/metric.pb.go

$(generated_libs): $(generate_sources)
	go generate ./...

vendor/${MESSAGE}/common/common.pb.go: vendor/${MESSAGE}/common.proto
	cd vendor/${MESSAGE} && go generate common/stub.go

vendor/${MESSAGE}/common.proto: vendor
	cp -r $(shell go list -mod=readonly -e -f '{{.Dir}}' ${MESSAGE})/common.proto vendor/${MESSAGE}/

vendor/${MESSAGE}/trident/trident.pb.go: vendor/${MESSAGE}/trident.proto
	cd vendor/${MESSAGE} && go generate trident/stub.go

vendor/${MESSAGE}/trident.proto: vendor
	cp -r $(shell go list -mod=readonly -e -f '{{.Dir}}' ${MESSAGE})/trident.proto vendor/${MESSAGE}/


libs/datatype/pb/flow_log.proto: vendor
	cp -r $(shell go list -mod=readonly -e -f '{{.Dir}}' ${MESSAGE})/flow_log.proto libs/datatype/pb/
	# delete first 3 char of line 6-10 for go language
	sed -i '6,10s/^...//' libs/datatype/pb/flow_log.proto

libs/datatype/pb/flow_log.pb.go: libs/datatype/pb/flow_log.proto
	cd libs/datatype/pb && go generate ./...

libs/zerodoc/pb/metric.proto: vendor
	cp -r $(shell go list -mod=readonly -e -f '{{.Dir}}' ${MESSAGE})/metric.proto libs/zerodoc/pb
	# delete first 3 char of line 6-10 for go language
	sed -i '6,10s/^...//' libs/zerodoc/pb/metric.proto

libs/zerodoc/pb/metric.pb.go: libs/zerodoc/pb/metric.proto
	cd libs/zerodoc/pb && go generate ./...

libs/geo/ip_info.go: libs/geo/ip_info_mini.json libs/geo/ip_info.py
	go generate ./...

vendor:
	go mod tidy && go mod download && go mod vendor
	find vendor -type d -exec chmod +w {} \;

.PHONY: test
test: vendor $(proto) $(generated_libs)
	go test -mod vendor -short ./... -timeout 5s -coverprofile .test-coverage.txt
	go tool cover -func=.test-coverage.txt

.PHONY: server
server: vendor $(proto) $(generated_libs)
	go build -mod vendor ${FLAGS} -o bin/metaflow-server${BINARY_SUFFIX} cmd/server/main.go

.PHONY: clean
clean:
	touch vendor
	chmod -R 777 vendor
	rm -rf vendor
	rm -rf bin
	rm -rf .test-coverage.txt
	find . -name '*.pb.go' -delete
	rm -f libs/geo/ip_info.go
	rm -f $(generated_libs)
	rm -f libs/datatype/pb/flow_log.proto
	rm -f libs/zerodoc/pb/metric.proto
