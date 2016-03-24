SOURCE = parser.go
CONTAINER = jsonparser
SOURCE_PATH = /go/src/github.com/buger/jsonparser
BENCHMARK = .
TEST = .

build:
	docker build -t $(CONTAINER) .

race:
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) --env GORACE="halt_on_error=1" go test ./. $(ARGS) -v -race -timeout 15s

bench:
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) go test $(LDFLAGS) -test.benchmem -bench $(BENCHMARK) ./benchmark/ $(ARGS) -timeout 15s -v

profile:
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) go test $(LDFLAGS) -test.benchmem -bench $(BENCHMARK) ./benchmark/ $(ARGS) -memprofile mem.mprof -v
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) go test $(LDFLAGS) -test.benchmem -bench $(BENCHMARK) ./benchmark/ $(ARGS) -cpuprofile cpu.out -v
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) go test $(LDFLAGS) -test.benchmem -bench $(BENCHMARK) ./benchmark/ $(ARGS) -c

test:
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) go test $(LDFLAGS) ./ -run $(TEST) -timeout 10s $(ARGS) -v

fmt:
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) go fmt ./...

vet:
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) go vet ./.


bash:
	docker run -v `pwd`:$(SOURCE_PATH) -i -t $(CONTAINER) /bin/bash