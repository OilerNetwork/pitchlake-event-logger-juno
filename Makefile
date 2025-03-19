ifeq ($(VM_DEBUG),true)
    GO_TAGS = -tags 'vm_debug,no_coverage'
    VM_TARGET = debug
else
    GO_TAGS = -tags 'no_coverage'
    VM_TARGET = all
endif

build:
	go build $(GO_TAGS) -a -ldflags="-X main.Version=$(shell git describe --tags)" -buildmode=plugin -o myplugin.so plugin/myplugin.go
