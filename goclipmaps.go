package goclipmaps

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/JoshVarga/svgparser"
	"github.com/engelsjk/geojson2svg"
	"github.com/engelsjk/geoviewport"
	"github.com/engelsjk/svgg"
	"github.com/fogleman/gg"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// Clipper ...
type Clipper struct{ MapboxAccessToken string }

// ReadFeature ...
func (c Clipper) ReadFeature(r io.Reader) (*geojson.Feature, error) {

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	f := &geojson.Feature{}
	err = f.UnmarshalJSON(b)
	if err != nil {
		return nil, err
	}

	switch f.Geometry.(type) {
	case *geom.Polygon, *geom.MultiPolygon:
		return f, nil
	default:
		return nil, fmt.Errorf("only Polygon or MultiPolygon geometries allowed")
	}
}

// Mask ...
type Mask struct {
	Feature *geojson.Feature
	W, H    float64
	Bounds  []float64
	Opt2x   bool
}

// NewMask ...
func (c Clipper) NewMask(feature *geojson.Feature) Mask {

	dimMaxPixels := 1280.0
	tileSize := 512

	bounds := featureBounds(feature)

	xDel := math.Abs(bounds[2] - bounds[0])
	yDel := math.Abs(bounds[3] - bounds[1])

	var w, h float64
	if xDel > yDel {
		w = dimMaxPixels
		h = w * yDel / xDel
	} else {
		h = dimMaxPixels
		w = h * xDel / yDel
	}

	dimensions := []float64{w, h}

	viewportCenter, viewportZoom := geoviewport.Viewport(bounds, dimensions, 0, 0, tileSize, true)
	viewportBounds := geoviewport.Bounds(viewportCenter, viewportZoom, dimensions, tileSize)

	return Mask{
		Feature: feature,
		W:       w,
		H:       h,
		Bounds:  viewportBounds,
		Opt2x:   true,
	}
}

// GetImage ...
func (c Clipper) GetImage(mask Mask) (image.Image, error) {

	urlStaticImage := "https://api.mapbox.com/styles/v1/mapbox/satellite-v9/static"

	size := fmt.Sprintf("%dx%d", int(mask.W), int(mask.H))
	focus := fmt.Sprintf("[%f,%f,%f,%f]", mask.Bounds[0], mask.Bounds[1], mask.Bounds[2], mask.Bounds[3])

	u, err := url.Parse(urlStaticImage)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, focus)
	u.Path = path.Join(u.Path, size)
	if mask.Opt2x {
		u.Path = fmt.Sprintf("%s@2x", u.Path)
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	queryParams := req.URL.Query()
	queryParams.Add("access_token", c.MapboxAccessToken)
	queryParams.Add("logo", "false")
	queryParams.Add("attribution", "false")

	req.URL.RawQuery = queryParams.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("image request error: %s", resp.Status)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return img, nil
}

// ClipAndSave ...
func (c Clipper) ClipAndSave(mask Mask, img image.Image, filename string) error {

	// resize mask to match @2x mapbox static image pixels
	if mask.Opt2x {
		mask.W = float64(2 * int(mask.W))
		mask.H = float64(2 * int(mask.H))
	}

	svg, err := featureToSVG(mask.Feature, mask.Bounds, mask.W, mask.H)
	if err != nil {
		return err
	}

	reader := strings.NewReader(string(svg))
	element, err := svgparser.Parse(reader, false)
	if err != nil {
		return err
	}

	dc := gg.NewContext(int(mask.W), int(mask.H))

	for _, child := range element.Children {

		if child.Name != "path" {
			continue
		}

		path := child.Attributes["d"]

		err = svgg.NewParser(dc).CompilePath(path)
		if err != nil {
			return err
		}
	}

	dc.Clip()
	dc.DrawImage(img, 0, 0)
	return dc.SavePNG(filename)
}

func featureBounds(f *geojson.Feature) []float64 {
	bounds := f.Geometry.Bounds()
	return []float64{bounds.Min(0), bounds.Min(1), bounds.Max(0), bounds.Max(1)}
}

// FeatureToSVG creates an SVG string from a GeoJSON feature
func featureToSVG(f *geojson.Feature, bounds []float64, w, h float64) (string, error) {

	b, err := f.MarshalJSON()
	if err != nil {
		return "", err
	}

	svg := geojson2svg.NewSVG()
	err = svg.AddFeature(string(b))
	if err != nil {
		return "", err
	}

	extent := &geojson2svg.Extent{MinX: bounds[0], MinY: bounds[1], MaxX: bounds[2], MaxY: bounds[3]}

	d := svg.Draw(w, h,
		geojson2svg.WithExtent(extent),
		geojson2svg.WithMercator(true),
	)

	return d, nil
}
