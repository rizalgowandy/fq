GO_BUILD_FLAGS ?= -trimpath
GO_BUILD_LDFLAGS ?= -s -w

all: test fq

# used to force make to always redo
.PHONY: always

fq: always
	CGO_ENABLED=0 go build -o fq -ldflags "${GO_BUILD_LDFLAGS}" ${GO_BUILD_FLAGS} .

test: always testgo testjq testcli
test-race: always testgo-race testjq testcli

# figure out all go packages with test files
testgo: PKGS=$(shell find . -name "*_test.go" | xargs -n 1 dirname | sort | uniq)
testgo: always
	go test -timeout 20m ${RACE} ${VERBOSE} ${COVER} ${PKGS}

testgo-race: RACE=-race
testgo-race: testgo

testjq: $(shell find . -name "*.jq.test")
%.jq.test: fq
	@echo $@
	@./fq -rRs -L pkg/interp 'include "jqtest"; run_tests' $@

testcli: fq
	@pkg/cli/test_exp.sh ./fq pkg/cli/test_repl.exp
	@pkg/cli/test_exp.sh ./fq pkg/cli/test_cli_ctrlc.exp
	@pkg/cli/test_exp.sh ./fq pkg/cli/test_cli_ctrld.exp

cover: COVER=-cover -coverpkg=./... -coverprofile=cover.out
cover: test
	go tool cover -html=cover.out -o cover.out.html
	cat cover.out.html | grep '<option value="file' | sed -E 's/.*>(.*) \((.*)%\)<.*/\2 \1/' | sort -rn

doc: always
doc: $(wildcard doc/*.svg.sh)
doc: $(wildcard *.md doc/*.md)

%.md: fq
	@doc/mdsh.sh ./fq $@

doc/%.svg.sh: fq
	(cd doc ; ../$@ ../fq) | go run github.com/wader/ansisvg@master > $(@:.svg.sh=.svg)

doc/formats.svg: fq
	@# ignore graphviz version as it causes diff when nothing has changed
	./fq -rnf doc/formats_diagram.jq | dot -Tsvg | sed 's/Generated by graphviz.*//' >doc/formats.svg

doc/file.mp3: Makefile
	ffmpeg -y -f lavfi -i sine -f lavfi -i testsrc -map 0:0 -map 1:0 -t 20ms "$@"

doc/file.mp4: Makefile
	ffmpeg -y -f lavfi -i sine -f lavfi -i testsrc -c:a aac -c:v h264 -f mp4 -t 20ms "$@"

gogenerate: always
	go generate -x ./...

lint: always
# bump: make-golangci-lint /golangci-lint@v([\d.]+)/ git:https://github.com/golangci/golangci-lint.git|^1
# bump: make-golangci-lint link "Release notes" https://github.com/golangci/golangci-lint/releases/tag/v$LATEST
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1 run

depgraph.svg: always
	go run github.com/kisielk/godepgraph@latest github.com/wader/fq | dot -Tsvg -o godepgraph.svg

# make memprof ARGS=". test.mp3"
# make cpuprof ARGS=". test.mp3"
prof: always
	go build -tags profile -o fq.prof .
	CPUPROFILE=fq.cpu.prof MEMPROFILE=fq.mem.prof ./fq.prof ${ARGS}
memprof: prof
	go tool pprof -http :5555 fq.prof fq.mem.prof
cpuprof: prof
	go tool pprof -http :5555 fq.prof fq.cpu.prof

update-gomod: always
	GOPROXY=direct go get -d github.com/wader/gojq@fq
	go mod tidy

# Usage: make fuzz # fuzz all foramts
# Usage: make fuzz GROUP=mp4 # fuzz a group (each format is a group also)
# TODO: as decode recovers panic and "repanics" unrecoverable errors this is a bit hacky at the moment
# Retrigger:
# try to decode crash with all formats in order to see which one panicked:
# cat format/testdata/fuzz/FuzzFormats/... | go run dev/fuzzbytes.go | go run . -d bytes '. as $b | formats | keys[] as $f | $b | decode($f)'
# convert crash into raw bytes:
# cat format/testdata/fuzz/FuzzFormats/... | go run dev/fuzzbytes.go | fq -d bytes to_base64
# fq -n '"..." | from_base64 | ...'
fuzz: always
# in other terminal: tail -f /tmp/repanic
	FUZZTEST=1 go test -v -run Fuzz -fuzz=Fuzz ./format/

# usage: make release VERSION=0.0.1
# tag forked dependeces for history and to make then stay around
release: always
release: WADER_GOJQ_COMMIT=$(shell go list -m -f '{{.Version}}' github.com/wader/gojq | sed 's/.*-\(.*\)/\1/')
release:
	@echo "# wader/fq":
	@echo "# make sure head is at wader/master"
	@echo git fetch wader
	@echo git show
	@echo make lint test doc
	@echo go mod tidy
	@echo git diff
	@echo
	@echo "sed 's/version = "\\\(.*\\\)"/version = \"${VERSION}\"/' fq.go > fq.go.new && mv fq.go.new fq.go"
	@echo git add fq.go
	@echo git commit -m \"fq: Update version to ${VERSION}\"
	@echo git push wader master
	@echo
	@echo "# make sure head master commit CI was successful"
	@echo open https://github.com/wader/fq/commit/master
	@echo git tag v${VERSION}
	@echo
	@echo "# wader/gojq:"
	@echo git tag fq-v${VERSION} ${WADER_GOJQ_COMMIT}
	@echo git push wader fq-v${VERSION}:fq-v${VERSION}
	@echo
	@echo "# wader/fq":
	@echo git push wader v${VERSION}:v${VERSION}
	@echo "# edit draft release notes and publish"


midi:
	go fmt ./format/midi/...
	go run . -d midi dv format/midi/testdata/test.mid

midi-test: fq
	go test ./format -run TestFormats/midi
