.PHONY: demo
demo:
	@cd cmd/revealcli && go run main.go --dir ../../data init demo --overwrite

.PHONY: start
start:
	@cd cmd/revealcli && go run main.go --dir ../../data start || true

.PHONY: build
build:
	@rm -rf data/build
	@cd cmd/revealcli && go run main.go --dir ../../data export --output ../../data/build

.PHONY: install
install:
	@cd cmd/revealcli && go install