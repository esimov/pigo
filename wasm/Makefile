ifeq ($(OS),Windows_NT)
    BROWSER = start
else
	UNAME := $(shell uname -s)
	ifeq ($(UNAME), Linux)
		BROWSER = xdg-open
	endif
	ifeq ($(UNAME), Darwin)
		BROWSER = open
	endif
endif

.PHONY: all clean serve

all: lib.wasm serve

%.wasm: %.go
	GOOS=js GOARCH=wasm go generate
	GOOS=js GOARCH=wasm go build -o "$@" "$<"

serve:
	$(BROWSER) 'http://localhost:5000'
	serve

clean:
	rm -f *.wasm

debug:
	@echo $(UNAME)