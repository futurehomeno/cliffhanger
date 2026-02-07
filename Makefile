define generate_mocks
    mockery --with-expecter --packageprefix mocked --keeptree --name=$(2) --recursive --case underscore --dir ./$(1) --output ./test/mocks/$(1)
endef

generate-mocks:
	find ./test/mocks -type f -not -name "*_helper.go" | xargs rm -rf
	$(call generate_mocks,"adapter/service","Reporter|Controller|Service|Manager")
	$(call generate_mocks,"adapter","Adapter|Thing|Connector|Service")
	$(call generate_mocks,"manifest","Loader")
	$(call generate_mocks,"storage","Storage")
	$(call generate_mocks,"prime","SyncClient")
	$(call generate_mocks,"root","Service|Resetter")
	$(call generate_mocks,"database","Database")
	find ./test/mocks -type f -name "thing_update.go" | xargs rm -rf #removes undesired mocks.

lint:
	golangci-lint run

test:
	go test -p 1 -v -covermode=atomic ./...

.PHONY: generate-mocks test lint
