all:
	# goimports -w *.go
	go build && ./chopsuey.exe |& pp
test:
	# goimports -w *.go
	go test |& pp
icon:
	# go get github.com/akavel/rsrc
	rsrc -ico chopsuey.ico
	# go build
.PHONY: all test
