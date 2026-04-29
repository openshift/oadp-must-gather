BRANCH ?= oadp-1.6
VERSION ?= 1.6.0
IMAGE ?= registry.redhat.io/oadp/oadp-mustgather-rhel9

CLI_FILE = pkg/cli.go

.PHONY: update-deps
update-deps:
	go get github.com/openshift/oadp-operator@$(BRANCH)
	go get github.com/migtools/oadp-non-admin@$(BRANCH)
	go get github.com/migtools/oadp-vm-file-restore@$(BRANCH)
	go mod tidy
	go mod verify

.PHONY: update-version
update-version:
ifndef VERSION
	$(error VERSION is required. Usage: make update-version VERSION=1.6)
endif
	sed 's|mustGatherVersion = ".*"|mustGatherVersion = "$(VERSION)"|' $(CLI_FILE) > $(CLI_FILE).tmp && mv $(CLI_FILE).tmp $(CLI_FILE)
	sed 's|mustGatherImage   = ".*"|mustGatherImage   = "$(IMAGE):v$(VERSION)"|' $(CLI_FILE) > $(CLI_FILE).tmp && mv $(CLI_FILE).tmp $(CLI_FILE)

.PHONY: prepare-release
prepare-release: update-deps update-version
