VERSION=0.6
all:
	echo "package main;const VERSION_STRING=\"v"${VERSION}" @ "`git log -1 --pretty="%h"`"\"" > version.go
	# go build && ./chopsuey.exe |& tee debug.log |& pp
release:
	echo "package main;const VERSION_STRING=\"v"${VERSION}"\"" > version.go

	# go get github.com/akavel/rsrc
	# rsrc -manifest chopsuey.exe.manifest -ico chopsuey.ico
	# go build -ldflags="-H windowsgui"
test:
	go test | pp
