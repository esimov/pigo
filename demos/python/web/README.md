## Webcam demo (slow)

This demo was prepared as a first attempt to show the Pigo library real time face detection capabilities and also as a POC for how it can be embedded into other languages and environments.

In this demo we are spinning up a Python application in order to access the webcam, then the captured frame rates are written to the standard output.

Next, in the Go application, we read the captured frames, convert them into a byte stream, run the face recognition algorithm on the captured data, and finally transform the result into a stream of images served on a web server.

### Run

Install the required Python dependencies, then run the following command:

```bash
$ go run main.go
```

Then access the `http://localhost:8081` url from a web browser.
