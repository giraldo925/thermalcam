package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"net/http"
	"time"

	"github.com/disintegration/gift"
	"github.com/sausheong/amg8833"
)

// used to interface with the sensor
var amg *amg8833.AMG88xx

// display frame
var frame string

// list of all colors used 1024 color hex integers
var colors []int

// the color image from the sensor 8x8 color hex integers
var pic []int

// temperature readings from the sensor 8x8 readings
var grid []float64

// frames per millisecond to capture and display the images
var fps *int

// minimum and maximum temperature range for the sensor
var minTemp, maxTemp *float64

// new image size in pixel width
var newSize *int

// if true, will use the mock data (this can be used for testing)
var mock *bool

func main() {
	// capture the user parameters from the command-line
	fps = flag.Int("f", 100, "frames per millisecond to capture and display the images")
	minTemp = flag.Float64("min", 26, "minimum temperature to measure from the sensor")
	maxTemp = flag.Float64("max", 32, "max temperature to measure from the sensor")
	newSize = flag.Int("s", 360, "new image size in pixel width")
	mock = flag.Bool("mock", false, "run using the mock data")
	flag.Parse()

	if *mock {
		// start populating the mock data into grid
		go startMock()
		fmt.Println("Using mock data.")
	} else {
		// start the thermal camera
		var err error
		amg, err = amg8833.NewAMG8833(&amg8833.Opts{
			Device: "/dev/i2c-1",
			Mode:   amg8833.AMG88xxNormalMode,
			Reset:  amg8833.AMG88xxInitialReset,
			FPS:    amg8833.AMG88xxFPS10,
		})
		if err != nil {
			panic(err)
		} else {
			fmt.Println("Connected to AMG8833 module.")
		}
		go startThermalCam()
	}

	// setting up the web server
	mux := http.NewServeMux()
	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	mux.HandleFunc("/", index)
	mux.HandleFunc("/frame", getFrame)
	server := &http.Server{
		Addr:    "0.0.0.0:12345",
		Handler: mux,
	}
	fmt.Println("Started AMG8833 Thermal Camera server at", server.Addr)
	server.ListenAndServe()

}

func index(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("public/index.html")
	// start generating frames in a new goroutine
	go generateFrames()
	t.Execute(w, 100)
}

// continually generate frames at every period
func generateFrames() {
	for {
		img := createImage(8, 8) // from 8 x 8 sensor
		createFrame(img)         // create the frame from the sensor
		time.Sleep(time.Duration(*fps) * time.Millisecond)
	}
}

// push the frame to the browser
func getFrame(w http.ResponseWriter, r *http.Request) {
	str := "data:image/png;base64," + frame
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(str))
}

// get the index of the color to usee
func getColorIndex(temp float64) int {
	if temp < *minTemp {
		return 0
	}
	return int((temp - *minTemp) * float64(len(colors)-1) / (*maxTemp - *minTemp))
}

// create a frame from the image
func createFrame(img image.Image) {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	frame = base64.StdEncoding.EncodeToString(buf.Bytes())
}

// create an enlarged image from the sensor
func createImage(w, h int) *image.RGBA {
	// create a RGBA image from the sensor
	pixels := image.NewRGBA(image.Rect(0, 0, w, h))
	n := 0
	for _, i := range grid {
		color := colors[getColorIndex(i)]
		pixels.Pix[n] = getR(color)
		pixels.Pix[n+1] = getG(color)
		pixels.Pix[n+2] = getB(color)
		pixels.Pix[n+3] = 0xFF // we don't need to use this
		n = n + 4
	}
	// now resize it
	g := gift.New(
		gift.Resize(*newSize, 0, gift.CubicResampling),
	)
	dest := image.NewRGBA(g.Bounds(pixels.Bounds()))
	g.Draw(dest, pixels)

	return dest
}

// get the red (R) from the color integer i
func getR(i int) uint8 {
	return uint8((i >> 16) & 0x0000FF)
}

// get the green (G) from the color integer i
func getG(i int) uint8 {
	return uint8((i >> 8) & 0x0000FF)
}

// get the blue (B) from the color integer i
func getB(i int) uint8 {
	return uint8(i & 0x0000FF)
}

// start the thermal camera and start getting sensor data into the grid
func startThermalCam() {
	for {
		grid = amg.ReadPixels()
		time.Sleep(time.Duration(*fps) * time.Millisecond)
	}
}
