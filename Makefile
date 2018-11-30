.PHONY: proto
proto:
	protoc --go_out=. testdata/testdata.proto
