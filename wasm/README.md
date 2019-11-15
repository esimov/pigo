## WASM (Webassembly) support

Thanks to the [syscall/js](https://golang.org/pkg/syscall/js/) package included into the Go base, Pigo has been ported to WASM 🎉. This gives a huge gain in terms of real time performance. 

This means that there is no need anymore to run it in a Python environment as a shared object library. For more details check the project description from the [Readme](https://github.com/esimov/pigo/blob/master/README.md#real-time-face-detection) file and also the [blog post](https://esimov.com/2019/11/pupilseyes-localization-in-the-pigo-face-detection-library).

## How to run it?

First download and build the [serve](https://github.com/mattn/serve) package for making a simple webserver. Then simply type the `$ serve` command to build the `wasm` file and start the webserver. That's all. It will open a new page under `http://localhost:5000/`.

In case the `lib.wasm` is not generated automatically you can build yourself running the following command:

```bash
$ GOOS=js GOARCH=wasm go build -o lib.wasm wasm.go
```

### Supported keys:
<kbd>s</kbd> - Show/hide pupils<br/>
<kbd>c</kbd> - Toggle circle/rectangle detection mark<br/>
