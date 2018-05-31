package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os/exec"
)

const boundary = "informs"

func main() {
	http.HandleFunc("/cam", cam)
	//http.HandleFunc("/test", test)
	log.Fatal(http.ListenAndServe(":8081", nil))
	//captureWebcam()
}

func cam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+boundary)
	cmd := exec.CommandContext(r.Context(), "./capture.py")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("[ERROR] Getting the stdout pipe")
		return
	}
	cmd.Start()

	mpart := multipart.NewReader(stdout, boundary)
	for {
		p, err := mpart.NextPart()
		if err == io.EOF {
			log.Println("[DEBUG] EOF")
			break
		}
		if err != nil {
			log.Println("[ERROR] reading next part", err)
			return
		}

		data, err := ioutil.ReadAll(p)
		if err != nil {
			log.Println("[ERROR] reading from bytes ", err)
			continue
		}
		img, _, _ := image.Decode(bytes.NewReader(data))

		frameBuffer := new(bytes.Buffer)
		err = jpeg.Encode(frameBuffer, img, nil)
		if err != nil {
			log.Println("[ERROR] encoding frame buffer ", err)
			continue
		}

		// cParams := pigo.CascadeParams{
		// 	MinSize:     100,
		// 	MaxSize:     1000,
		// 	ShiftFactor: 0.1,
		// 	ScaleFactor: 1.1,
		// }

		// imgParams := pigo.ImageParams{sampleImg, rows, cols, cols}

		// pigo := pigo.NewPigo()
		// // Unpack the binary file. This will return the number of cascade trees,
		// // the tree depth, the threshold and the prediction from tree's leaf nodes.
		// classifier := pigo.Unpack("../data/facefinder")

		// // Run the classifier over the obtained leaf nodes and return the detection results.
		// // The result contains quadruplets representing the row, column, scale and detection score.
		// dets := classifier.RunCascade(imgParams, cParams)

		// // Calculate the intersection over union (IoU) of two clusters.
		// dets = classifier.ClusterDetections(dets, 0)
		// just MJPEG
		w.Write([]byte("Content-Type: image/jpeg\r\n"))
		w.Write([]byte("Content-Length: " + string(len(data)) + "\r\n\r\n"))
		//w.Write(frameBuffer.Bytes())
		w.Write(frameBuffer.Bytes())
		w.Write([]byte("\r\n"))
		w.Write([]byte("--informs\r\n"))
	}
	cmd.Wait()
}

func test(w http.ResponseWriter, r *http.Request) {
	// set the multipart header
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+boundary)
	// execute capture.py with the context
	cmd := exec.CommandContext(r.Context(), "./capture.py")
	// connect the stdout from capture to response writer
	cmd.Stdout = w

	err := cmd.Run()
	if err != nil {
		log.Println("[ERROR] capturing webcam", err)
	}
}

func captureWebcam() {
	cmd := exec.Command("./capture.py")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("[ERROR] Getting the stdout pipe")
		return
	}
	cmd.Start()

	mr := multipart.NewReader(stdout, boundary)
	for {
		p, err := mr.NextPart()

		if err == io.EOF {
			log.Println("[DEBUG] EOF")
			break
		}
		if err != nil {
			log.Println("[ERROR] reading next part", err)
			return
		}
		jp, err := ioutil.ReadAll(p)

		if err != nil {
			log.Println("[ERROR] reading from bytes ", err)
			continue
		}
		//jpReader := bytes.NewReader(jp)

		fmt.Println(jp)
	}
	cmd.Wait()
}
