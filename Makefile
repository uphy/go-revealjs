.PHONY: demo
demo:
	@cd cmd/revealgo && go run main.go --dir ../../data init demo --overwrite

.PHONY: start
start:
	@cd cmd/revealgo && go run main.go --dir ../../data start || true

.PHONY: build
build:
	@cd cmd/revealgo && go run main.go --dir ../../data build

.PHONY: install
install:
	@cd cmd/revealgo && go install