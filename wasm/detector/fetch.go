//go:build js && wasm
// +build js,wasm

package detector

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"syscall/js"
)

// Detector struct holds the main components of the fetching operation.
type Detector struct {
	respChan chan []uint8
	errChan  chan error
	done     chan struct{}

	window js.Value
}

// NewDetector initializes a new constructor function.
func NewDetector() *Detector {
	var d Detector
	d.window = js.Global()

	return &d
}

// FetchCascade retrive the cascade file through a JS http connection.
// It should return the binary data as uint8 integers or err in case of an error.
func (d *Detector) FetchCascade(url string) ([]byte, error) {
	d.respChan = make(chan []uint8)
	d.errChan = make(chan error)

	promise := js.Global().Call("fetch", url)
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			response := args[0]
			if !response.Get("ok").Bool() {
				errorMsg := response.Get("statusText").String()
				d.errChan <- errors.New(errorMsg)
			}
		}()
		return nil
	}))
	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		response.Call("arrayBuffer").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			go func() {
				buffer := args[0]
				uint8Array := js.Global().Get("Uint8Array").New(buffer)

				jsbuf := make([]byte, uint8Array.Get("length").Int())
				js.CopyBytesToGo(jsbuf, uint8Array)
				d.respChan <- jsbuf
			}()
			return nil
		}))
		return nil
	})

	failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			err := fmt.Errorf("unable to fetch the cascade file: %s", args[0].String())
			d.errChan <- err
		}()
		return nil
	})

	promise.Call("then", success, failure)

	select {
	case resp := <-d.respChan:
		return resp, nil
	case err := <-d.errChan:
		return nil, err
	}
}

// ParseCascade loads and parse the cascade file through the
// Javascript `location.href` method, using the `js/syscall` package.
// It will return the cascade file encoded into a byte array.
func (d *Detector) ParseCascade(path string) ([]byte, error) {
	href := js.Global().Get("location").Get("href")
	u, err := url.Parse(href.String())
	if err != nil {
		return nil, err
	}
	u.Path = path

	resp, err := http.Get(u.String())
	if err != nil || resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%v cascade file is missing", u.String()))
	}
	defer resp.Body.Close()

	buffer, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	uint8Array := js.Global().Get("Uint8Array").New(len(buffer))
	js.CopyBytesToJS(uint8Array, buffer)

	buff := make([]byte, uint8Array.Get("length").Int())
	js.CopyBytesToGo(buff, uint8Array)

	return buff, nil
}

// Log calls the `console.log` Javascript function
func (d *Detector) Log(args ...interface{}) {
	d.window.Get("console").Call("log", args...)
}
