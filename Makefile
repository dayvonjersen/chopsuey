all:
	go build && ./chopsuey.exe |& pp
release:
	# go get github.com/akavel/rsrc
	rsrc -manifest chopsuey.exe.manifest -ico chopsuey.ico
	go build -ldflags="-H windowsgui"
test:
	go test | pp
