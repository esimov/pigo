package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
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
		data, err := ioutil.ReadAll(p)
		if err != nil {
			log.Println("[ERROR] reading from bytes ", err)
			continue
		}
		img, _, _ := image.Decode(bytes.NewReader(data))

		//jpReader := bytes.NewReader(jp)
		//fmt.Println(jp)
		frameBuffer := new(bytes.Buffer)
		bw := bufio.NewWriter(w)

		width, height := img.Bounds().Dx(), img.Bounds().Dy()
		target := image.NewNRGBA(image.Rect(0, 0, width, height))
		c := color.NRGBA{R: uint8(0), G: uint8(0), B: uint8(0), A: uint8(255)}

		for y := 0; y < height; y++ {
			for x := 0; x < width; x += 4 {
				fmt.Println(y*width + x)
				fmt.Println("LEN:", len(data))
				//fmt.Println(data[y*width+x])
				c.R = uint8(data[y*width+x+0])
				c.G = uint8(data[y*width+x+1])
				c.B = uint8(data[y*width+x+2])
				c.A = uint8(data[y*width+x+3])

				target.SetNRGBA(int(x/4), y, c)
			}
		}

		bw.Write(frameBuffer.Bytes())
		jpeg.Encode(frameBuffer, target, nil)
		fmt.Println(target)
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
