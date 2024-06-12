.PHONY: demo
demo:
	@cd cmd/revealcli && go run . --dir ../../data init demo --overwrite

.PHONY: start
start:
	@cd cmd/revealcli && go run . --dir ../../data start || true

.PHONY: build
build:
	@rm -rf data/build
	@cd cmd/revealcli && go run . --dir ../../data export --output ../../data/build

.PHONY: test
test:
	@go test -v ./...

.PHONY: install
install:
	@cd cmd/revealcli && go install