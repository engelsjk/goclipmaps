package main

import (
	"flag"
	"log"
	"os"

	"github.com/engelsjk/goclipmaps"
	"github.com/joho/godotenv"
)

var MapboxAccessToken string

func init() {
	MapboxAccessToken = os.Getenv("MAPBOX_ACCESS_TOKEN")
	if MapboxAccessToken != "" {
		return
	}
	if err := godotenv.Load(); err != nil {
		log.Fatal("MAPBOX_ACCESS_TOKEN environmental variable required")
	}
}

func main() {
	run()
}

func run() {

	var input = flag.String("shape", "shape.geojson", "input geojson file")
	var output = flag.String("o", "clip.png", "output clip image")

	flag.Parse()

	//////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////////////////////////

	clipper := goclipmaps.Clipper{MapboxAccessToken: MapboxAccessToken}

	file, err := os.Open(*input)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	feature, err := clipper.ReadFeature(file)
	if err != nil {
		log.Fatal(err)
	}

	mask := clipper.NewMask(feature)

	img, err := clipper.GetImage(mask)
	if err != nil {
		log.Fatal(err)
	}

	err = clipper.ClipAndSave(mask, img, *output)
	if err != nil {
		log.Fatal(err)
	}
}
