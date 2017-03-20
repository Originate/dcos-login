PACKAGES?=$$(glide novendor)

default: test

tools:
	go get -u github.com/Masterminds/glide
	go get -u github.com/golang/lint/golint

# dev creates binaries for testing locally. These are put
# into ./bin/ as well as $GOPATH/bin
dev: fmtcheck
	go install $(PACKAGES)

# test runs the unit tests with race detection
test: fmtcheck
	go test -race $(PACKAGES) $(TESTARGS)

# vet runs the Go source code static analysis tool `vet` to find
# any common errors.
vet:
	@echo "go tool vet  ."
	@go tool vet -all $$(ls -d */ | grep -v vendor) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

# lint runs the Go source code static analysis tool `golint` to find
# any common style errors.
lint:
	@for PACKAGE in $$(find . -iname '*.go' -exec dirname {} \; | grep -v 'vendor' | sort | uniq) ; do \
		golint $$(find $$PACKAGE -name '*.go' -maxdepth 1 | grep -v 'test.go'); \
	done

fmt:
	go fmt $(PACKAGES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/ci/fmt-check'"

.PHONY: test vet lint fmt fmtcheck tools
