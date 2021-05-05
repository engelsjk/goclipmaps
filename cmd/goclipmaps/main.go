package main

import (
	"flag"
	"log"
	"os"

	"github.com/engelsjk/goclipmaps"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error loading .env file")
	}
}

func main() {
	run()
}

func run() {

	var input = flag.String("shape", "shape.geojson", "input geojson file")
	var output = flag.String("clip", "clip.png", "output clip image")

	flag.Parse()

	//////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////////////////////////

	clipper := goclipmaps.Clipper{MapboxAccessToken: os.Getenv("MAPBOX_ACCESS_TOKEN")}

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
