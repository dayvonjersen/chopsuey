VERSION=0.7
all:
	@echo "package main;const VERSION_STRING=\"v"${VERSION}" @ "`git log -1 --pretty="%h"`"\"" > version.go
	go build && ./chopsuey.exe |& tee ./.log/`date +'%Y%m%d%H%M%S.%N'`.log |& pp
release: 
	@echo "package main;const VERSION_STRING=\"v"${VERSION}"\"" > version.go

	go build -ldflags="-H windowsgui"
test:
	go test | pp
icon:
	# go get github.com/akavel/rsrc
	rsrc -manifest chopsuey.exe.manifest -ico chopsuey.ico
