all:
	go build && ./chopsuey.exe |& pp
release:
	go build -ldflags="-H windowsgui"
test:
	go test |& pp
icon:
	# go get github.com/akavel/rsrc
	rsrc -ico chopsuey.ico
richedit:
	go build richedit.go && ./richedit.exe |&pp
