define generate_mocks
    mockery --packageprefix mocked --keeptree --name=$(2) --recursive --case underscore --dir ./$(1) --output ./test/mocks/$(1)
endef

.phony: generate-mocks
generate-mocks:
	find ./test/mocks -type f -not -name "*_helper.go" | xargs rm -rf
	$(call generate_mocks,"adapter/service","Reporter|Controller|Service")
	$(call generate_mocks,"adapter","Adapter|Thing")
	$(call generate_mocks,"manifest","Loader")
	$(call generate_mocks,"storage","Storage")
