## Pigo demos running in a Python environment

This directory contains a few real time demos running as shared objects (**.so**) in a Python environment. The face detection is happening on the Go side but the results are transfered as a byte array to the Python program. It was intended this way because the Go ecosystem is still missing a cross platform and operating system agnostic webcam library. This dependency issue is partially resolved with the Webassembly (WASM) port of the library. 

## Requirements
- OpenCV 2
- Python2

## Notice

For the `WASM` port check this subfolder:

https://github.com/esimov/pigo/tree/master/wasm
