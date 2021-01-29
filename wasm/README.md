## WASM (Webassembly) integration

**Important notice: in order to run the Webassembly demo at least Go 1.13 is required!**

Starting from **v1.4.0** Pigo has been ported to Webassembly 🎉. This proves the library real time performance capabilities.

This also means that it's not needed anymore to run the library in a Python environment as a shared library for real time face detection. For more details check the project description from the [Readme](https://github.com/esimov/pigo/blob/master/README.md#real-time-face-detection) file and also read this [blog post](https://esimov.com/2019/11/pupilseyes-localization-in-the-pigo-face-detection-library) for a more detailed explanation.

## How to run it?

In order to run the WASM demo is as simple as to type the <kbd>make</kbd> command inside the `wasm` folder. This will build the `wasm` file and will start a new webserver. That's all. It will open a new page in your browser under the following address: `http://localhost:5000/`.

In case the `lib.wasm` is not getting generated automatically you can build it yourself by running the following command:

```bash
$ GOOS=js GOARCH=wasm go build -o lib.wasm wasm.go
```
### Supported keys:
<kbd>s</kbd> - Show/hide pupils<br/>
<kbd>c</kbd> - Circle through the detection shape types (`rectangle`|`circle`|`ellipse`)<br/>
<kbd>f</kbd> - Show/hide facial landmark points (hidden by default)

## Demos

For **Webassembly** related demos using the **Pigo** library check this separate repo:

https://github.com/esimov/pigo-wasm-demos

## License

Copyright © 2019 Endre Simo
