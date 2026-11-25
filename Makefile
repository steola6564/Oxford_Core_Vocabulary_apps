GO ?= go

.PHONY: build run test verify

build:
	CGO_ENABLED=1 $(GO) build -buildvcs=false -mod=vendor -trimpath -tags sqlite_omit_load_extension -o bin/oxford-core .

run:
	CGO_ENABLED=1 $(GO) run -buildvcs=false -mod=vendor -tags sqlite_omit_load_extension .

test:
	CGO_ENABLED=1 $(GO) test -buildvcs=false -mod=vendor ./...

verify: build
	python3 tests/verify_api.py --binary ./bin/oxford-core --corpus ./data/all_with_ID.csv --output ./test-results/api-verification.json
