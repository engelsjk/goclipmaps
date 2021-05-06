package main

import (
	"flag"
	"log"
	"os"

	"github.com/engelsjk/goclipmaps"
	"github.com/joho/godotenv"
)

func main() {
	run()
}

func run() {

	var input = flag.String("shape", "shape.geojson", "input geojson file")
	var output = flag.String("o", "clip.png", "output clip image")

	flag.Parse()

	//////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////////////////////////

	token := getToken()
	if token == "" {
		log.Fatal("envvar or .env with MAPBOX_ACCESS_TOKEN is required")
	}

	clipper := goclipmaps.Clipper{MapboxAccessToken: token}

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

func getToken() string {
	if _, ok := os.LookupEnv("MAPBOX_ACCESS_TOKEN"); !ok {
		if err := godotenv.Load(); err != nil {
			return ""
		}
	}
	token, ok := os.LookupEnv("MAPBOX_ACCESS_TOKEN")
	if !ok {
		return ""
	}
	return token
}
