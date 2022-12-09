REPO ?= openyurt

# Unify ports of the edgex component
UNIFIED_PORT ?= 2000
DEBUG ?= false

.PHONY: run
run:
	go run main.go --debug=$(DEBUG) --unified-port=$(UNIFIED_PORT)

sync_image:
	chmod +x ./hack/sync_images.sh
	./hack/sync_images.sh $(REPO)