# Unify ports of the edgex component
UNIFIED_PORT ?= 2000
DEBUG ?= false

.PHONY: run
run:
	go run main.go --debug=$(DEBUG) --unified-port=$(UNIFIED_PORT)

CONTENT = $(shell cat ./../../EdgeXConfig/image.yaml)
IMAGES = $(subst - , ,$(CONTENT))
FIN_IMAGES = $(strip $(IMAGES))

IMG ?= yangbobo/consul:1.10.10

docker-push-mutiarch:
	#IMG = $(foreach image, $(FIN_IMAGES), $(image))
	docker buildx build --platform linux/arm64,linux/amd64 -t ${IMG} . --push