.PHONY: all build run gotool clean help

APPNAME=pine-ai

build:
	go build -o bin/${APPNAME} main.go

run:
	bin/${APPNAME}

clean:
	@if [ -f bin/${APPNAME} ] ; then rm bin/${APPNAME} ; fi