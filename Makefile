# golangci-lint v2.13.2 is built with go1.27.0; pinning the toolchain keeps
# its type-checker able to read this module's export data.
export GOTOOLCHAIN ?= go1.27.0

CUSTOM_GCL ?= ./custom-gcl

# Evaluated when make starts, before the $(CUSTOM_GCL) target can build it,
# so the warning below only fires for a binary that already existed.
CUSTOM_GCL_PREBUILT := $(wildcard $(CUSTOM_GCL))

.PHONY: lint lint-fix fmt test fuzz custom-gcl

lint: $(CUSTOM_GCL)
	@if [ -n "$(CUSTOM_GCL_PREBUILT)" ]; then \
		echo "WARNING: $(CUSTOM_GCL) already existed; make only rebuilds it when .custom-gcl.yml changes."; \
		echo "If pkg/ or plugin/ changed since it was built, run: rm $(CUSTOM_GCL) && make lint"; \
	fi
	$(CUSTOM_GCL) run

lint-fix: $(CUSTOM_GCL)
	@if [ -n "$(CUSTOM_GCL_PREBUILT)" ]; then \
		echo "WARNING: $(CUSTOM_GCL) already existed; make only rebuilds it when .custom-gcl.yml changes."; \
		echo "If pkg/ or plugin/ changed since it was built, run: rm $(CUSTOM_GCL) && make lint-fix"; \
	fi
	$(CUSTOM_GCL) run --fix

fmt:
	golangci-lint fmt

test:
	go test ./...

# FUZZTIME can be raised for longer sessions, e.g. make fuzz FUZZTIME=5m.
fuzz:
	go test -fuzz=FuzzAnalyzer -fuzztime=$(FUZZTIME) ./pkg
	go test -fuzz=FuzzFixpoint -fuzztime=$(FUZZTIME) ./pkg

# Builds golangci-lint with the easyreader plugin (clones upstream once).
$(CUSTOM_GCL): .custom-gcl.yml
	golangci-lint custom
