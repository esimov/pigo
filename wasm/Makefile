ifeq ($(OS),Windows_NT)
    BROWSER = start
else
    BROWSER = xdg-open
endif

.PHONY: all clean serve

all: main.wasm serve

%.wasm: %.go
	GOOS=js GOARCH=wasm go generate
	GOOS=js GOARCH=wasm go build -o "$@" "$<"

serve:
	$(BROWSER) 'http://localhost:5000'
	serve

clean:
	rm -f *.wasm