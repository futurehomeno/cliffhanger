define generate_mocks
    mockery --packageprefix mocked --keeptree --name=$(2) --recursive --case underscore --dir ./$(1) --output ./test/mocks/$(1)
endef

.phony: generate-mocks
generate-mocks:
	find ./test/mocks -type f -not -name "*_helper.go" | xargs rm -rf
	$(call generate_mocks,"adapter/service","Reporter|Controller|Service|Manager")
	$(call generate_mocks,"adapter","Adapter|Thing|Connector|Service")
	$(call generate_mocks,"manifest","Loader")
	$(call generate_mocks,"storage","Storage")
	$(call generate_mocks,"prime","SyncClient")
	$(call generate_mocks,"root","Service|Resetter")
	$(call generate_mocks,"database","Database")

lint:
	golangci-lint run

tests:
	docker container rm -f mqtt
	docker-compose up -d mqtt
	go test -p 1 -v -covermode=atomic ./...