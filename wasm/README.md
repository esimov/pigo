## Run

Start the WASM demo by simply running the <kbd>make</kbd> command inside the `wasm` folder. This will build the `wasm` file and at the same time will start a new webserver. That's all. It will automatically open a new page in your default browser under the following address: `http://localhost:5000/`.

In case the `lib.wasm` is not getting generated automatically you can build it yourself by running the following command:

```bash
$ GOOS=js GOARCH=wasm go build -o lib.wasm wasm.go
```
### Supported keys:
<kbd>s</kbd> - Show/hide pupils<br/>
<kbd>c</kbd> - Circle through the detection shape types (`rectangle`|`circle`|`ellipse`)<br/>
<kbd>f</kbd> - Show/hide facial landmark points (hidden by default)

## Demos

For showcasing WASM related demos using the Pigo library check this separate repo:

https://github.com/esimov/pigo-wasm-demos

## License

Copyright © 2019 Endre Simo
