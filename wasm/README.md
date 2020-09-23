## WASM (Webassembly) support

**Important note: in order to run the Webassembly demo at least Go 1.13 is required!**

Starting from **v1.4.0** Pigo has been ported to Webassembly 🎉. This gives it a huge gain in terms of real time performance.

This means also that there is no need to run the library in a Python environment as a shared library. For more details check the project description from the [Readme](https://github.com/esimov/pigo/blob/master/README.md#real-time-face-detection) file and also read this [blog post](https://esimov.com/2019/11/pupilseyes-localization-in-the-pigo-face-detection-library) for a more detailed explanation.

## How to run it?

Download and build the [serve](https://github.com/mattn/serve) package to make a simple webserver. Then simply type `$ make` to build the `wasm` file and to start the webserver. That's all. It will open a new page in your browser under the following address: `http://localhost:5000/`.

In case the `lib.wasm` is not generated automatically you can build it yourself by running the following command:

```bash
$ GOOS=js GOARCH=wasm go build -o lib.wasm wasm.go
```
### Supported keys:
<kbd>s</kbd> - Show/hide pupils<br/>
<kbd>c</kbd> - Circle trough the detection shape types (`rectangle`|`circle`|`ellipse`)<br/>
<kbd>f</kbd> - Show/hide facial landmark points (hidden by default)

## Demos

For **Webassembly** related demos check this separate repo: 

https://github.com/esimov/pigo-wasm-demos 

