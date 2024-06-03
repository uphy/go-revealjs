.PHONY: demo
demo:
	@cd cmd/revealcli && go run main.go --dir ../../data init demo --overwrite

.PHONY: start
start:
	@cd cmd/revealcli && go run main.go --dir ../../data start || true

.PHONY: build
build:
	@cd cmd/revealcli && go run main.go --dir ../../data build

.PHONY: install
install:
	@cd cmd/revealcli && go install