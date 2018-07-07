all:
	go build && ./chopsuey.exe |& pp
release:
	rsrc -manifest chopsuey.exe.manifest -ico chopsuey.ico
	go build -ldflags="-H windowsgui"
test:
	go test |& pp
icon:
	# go get github.com/akavel/rsrc
	rsrc -ico chopsuey.ico
richedit:
	go build -o richedit.exe richedit.go colors.go && ./richedit.exe |&pp
