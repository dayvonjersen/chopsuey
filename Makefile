all:
	goimports -w *.go
	go build && ./client.exe |& pp
test:
	goimports -w *.go
	go test |& pp
.PHONY: all test
