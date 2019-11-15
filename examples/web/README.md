## Webcam demo (slow)

This demo shows how we can transfer the webcam frames from Python to Go by sending the captured frames as byte array into the standard output. 
We will run the face detector over the byte arrays received from the standard output and send the result into a web browser trough a webserver.

### Run

```bash
$ go run main.go -cf "../../cascade/facefinder"
```

Then access the `http://localhost:8081/cam` url from a web browser.
