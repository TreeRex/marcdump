.PHONY: all debug

DEBUG_FLAGS = -gcflags "-N -l"

all:
	go clean
	go build

debug:
	go clean
	go build $(DEBUG_FLAGS)
