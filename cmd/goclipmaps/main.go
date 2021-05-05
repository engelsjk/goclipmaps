package main

import (
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

	shapeFilename := "test/shapes/multipolygon.geojson"
	clipFilename := "test/clip.png"

	clipper := goclipmaps.Clipper{MapboxAccessToken: os.Getenv("MAPBOX_ACCESS_TOKEN")}

	file, err := os.Open(shapeFilename)
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

	err = clipper.Clip(mask, img, clipFilename)
	if err != nil {
		log.Fatal(err)
	}
}
