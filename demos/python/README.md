## Pigo demos running in a Python environment

This directory contains a few real time face detection demos running as shared library (**.so**) in a Python environment. The face detection is done on the Go side, but the detected results are transfered as 1D array over the Python program. It was intended this way because the Go ecosystem is still missing a cross platform and operating system agnostic webcam library. This dependency issue is partially resolved with the Webassembly (WASM) port of the library.

## Requirements
- OpenCV2
- Python

## Notice

For the `WASM` port check this subfolder:

https://github.com/esimov/pigo/tree/master/wasm
