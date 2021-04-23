package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/JoshVarga/svgparser"
	"github.com/engelsjk/geojson2svg"
	"github.com/engelsjk/geoviewport"
	"github.com/engelsjk/svgg"
	"github.com/fogleman/gg"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/xy"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error loading .env file")
	}
}

const (
	maxPx    = 1280
	tileSize = 512
)

// ClipData ...
type ClipData struct {
	Feature    geojson.Feature
	Viewport   geoviewport.VP
	W, H       float64
	Bounds     []float64
	Center     []float64
	Dimensions []float64
	Opt2x      bool
}

func main() {
	d := NewClipData("shapes/34023002805.geojson")
	Clip(d)
}

// NewClipData ...
func NewClipData(fn string) ClipData {

	b := LoadFile(fn)
	var f geojson.Feature
	err := f.UnmarshalJSON(b)
	if err != nil {
		log.Fatal(err)
	}

	featureBounds := FeatureBounds(f)
	center := FeatureCenter(f)

	xDel := math.Abs(featureBounds[2] - featureBounds[0])
	yDel := math.Abs(featureBounds[3] - featureBounds[1])

	var w, h float64
	if xDel > yDel {
		w = maxPx
		h = w * yDel / xDel
	} else {
		h = maxPx
		w = h * xDel / yDel
	}

	dimensions := []float64{w, h}

	viewport := geoviewport.Viewport(featureBounds, dimensions, 0, 0, tileSize, true)
	bounds := geoviewport.Bounds(viewport.Center, viewport.Zoom, dimensions, tileSize)

	// fmt.Printf("featureBounds: %# v\n", pretty.Formatter(featureBounds))
	// fmt.Printf("bounds: %# v\n", pretty.Formatter(bounds))
	// fmt.Printf("center: %# v\n", pretty.Formatter(center))
	// fmt.Printf("vp: %# v\n", pretty.Formatter(vp))
	// fmt.Printf("dimensions: %# v\n", pretty.Formatter(dimensions))
	// fmt.Printf("w,h: %f,%f\n", w, h)

	return ClipData{
		Feature:    f,
		Viewport:   viewport,
		W:          w,
		H:          h,
		Bounds:     bounds,
		Center:     center,
		Dimensions: dimensions,
		Opt2x:      true,
	}
}

// Clip ...
func Clip(d ClipData) {

	/////////////////////////////////////////////////
	// get static map image

	img := GetImage(d.Bounds, d.W, d.H, d.Opt2x)

	/////////////////////////////////////////////////
	// create svg from feature

	if d.Opt2x {
		d.W = float64(2 * int(d.W))
		d.H = float64(2 * int(d.H))
	}

	svg := FeatureToSVG(d.Feature, img, d.Bounds, d.W, d.H)

	/////////////////////////////////////////////////
	// parse svg to get 1st path

	reader := strings.NewReader(string(svg))
	element, err := svgparser.Parse(reader, false)
	if err != nil {
		log.Fatal(err)
	}
	path := element.Children[0].Attributes["d"]

	/////////////////////////////////////////////////
	// clip & save

	dc := gg.NewContext(int(d.W), int(d.H))

	p := svgg.NewParser(dc)
	p.CompilePath(path)

	dc.Clip()
	dc.DrawImage(img, 0, 0)
	dc.SavePNG("clip.png")
}

// LoadFile ...
func LoadFile(f string) []byte {

	file, err := os.Open(f)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

// FeatureToSVG creates an SVG string from a GeoJSON feature
func FeatureToSVG(f geojson.Feature, img image.Image, bounds []float64, w, h float64) string {

	b, err := f.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}

	svg := geojson2svg.NewSVG()
	err = svg.AddFeature(string(b))
	if err != nil {
		log.Fatal(fmt.Errorf("unexpected error %v", err))
	}

	extent := &geojson2svg.Extent{MinX: bounds[0], MinY: bounds[1], MaxX: bounds[2], MaxY: bounds[3]}

	d := svg.Draw(w, h,
		geojson2svg.WithExtent(extent),
		geojson2svg.WithMercator(true),
	)

	return d
}

// GetImage ...
func GetImage(bounds []float64, w, h float64, opt2x bool) image.Image {

	urlStaticImage := "https://api.mapbox.com/styles/v1/mapbox/satellite-v9/static"

	size := fmt.Sprintf("%dx%d", int(w), int(h))
	focus := fmt.Sprintf("[%f,%f,%f,%f]", bounds[0], bounds[1], bounds[2], bounds[3])

	u, err := url.Parse(urlStaticImage)
	if err != nil {
		log.Fatal(err)
	}

	u.Path = path.Join(u.Path, focus)
	u.Path = path.Join(u.Path, size)
	if opt2x {
		u.Path = fmt.Sprintf("%s@2x", u.Path)
	}

	// fmt.Printf("url: %# v\n", pretty.Formatter(u.String()))

	queryParams := map[string]string{
		"access_token": os.Getenv("MAPBOX_ACCESS_TOKEN"),
	}

	client := resty.New()
	resp, err := client.R().
		SetQueryParams(queryParams).
		Get(u.String())
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode() != 200 {
		log.Fatal(resp.Status())
	}

	img, _, err := image.Decode(bytes.NewReader(resp.Body()))
	if err != nil {
		log.Fatal(err)
	}

	return img
}

// FeatureBounds ...
func FeatureBounds(f geojson.Feature) []float64 {
	bounds := f.Geometry.Bounds()
	return []float64{bounds.Min(0), bounds.Min(1), bounds.Max(0), bounds.Max(1)}
}

// FeatureCenter ...
func FeatureCenter(f geojson.Feature) []float64 {
	b := f.Geometry.Bounds().Polygon()
	centroid, _ := xy.Centroid(b)
	return []float64{centroid.X(), centroid.Y()}
}
