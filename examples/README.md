## Python demos

This directory contains a few real time Python demos running as shared object (**.so**). The face detection is happing on the Go side and the results are transfered as byte array to Python as a shared object. It was intended this way since there is a huge hiatus in the Go echosystem of a throughly accessible and platform agnostic webcam library. This dependency issue is partially resolved with the Webassembly (WASM) port of the library. 


## Requirements
- OpenCV 2
- Python2

## Notice

For the **WASM** port check this subfolder:

https://github.com/esimov/pigo/tree/master/wasm
