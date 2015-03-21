package geojson

import (
	"errors"
	"fmt"
)

// lowest and highest values fo coordinates
type BoundingBox []float64

// A GeoJSON object with the type "Feature" is a feature object.
// - A feature object must have a member with the name "geometry".
//	 The value of the geometry member is a geometry object or a JSON null value.
// - A feature object must have a member with the name "properties".
//   The value of the properties member is an object (any JSON object
//   or a JSON null value).
// - If a feature has a commonly used identifier, that identifier should be
//   included as a member of the feature object with the name "id".
type Feature struct {
	Type       string                 `json:"type"`
	Id         interface{}            `json:"id,omitempty"`
	Geometry   interface{}            `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
	Bbox       BoundingBox            `json:"bbox,omitempty"`
	Crs        *CRS                   `json:"crs,omitempty"`
}

func (t *Feature) GetGeometry() (Geometry, error) {
	gi := t.Geometry
	return parseGeometry(gi)
}

// Factory constructor method
func NewFeature(geom Geometry, properties map[string]interface{}, id interface{}) *Feature {
	return &Feature{Type: "Feature",
		Geometry:   geom,
		Properties: properties,
		Id:         id}
}

// An object of type "FeatureCollection" must have a member with the name
// "features". The value corresponding to "features" is an array.
// Each element in the array is a Feature object.
type FeatureCollection struct {
	Type     string      `json:"type"`
	Features []*Feature  `json:"features"`
	Bbox     BoundingBox `json:"bbox,omitempty"`
	Crs      *CRS        `json:"crs,omitempty"`
}

func (t *FeatureCollection) AddFeatures(f ...*Feature) {
	if f == nil {
		t.Features = make([]*Feature, 0, 100)
	}
	t.Features = append(t.Features, f...)
}

// factory funcion
func NewFeatureCollection(features []*Feature) *FeatureCollection {
	return &FeatureCollection{Type: "FeatureCollection", Features: features}
}

// The coordinate reference system (CRS) of a GeoJSON object
// is determined by its "crs" member.
type CRS struct {
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties"`
}

//Example:
//"crs": {
//  "type": "name",
//  "properties": {
//    "name": "urn:ogc:def:crs:OGC:1.3:CRS84"
//    }
//  }
func NewNamedCRS(name string) *CRS {
	return &CRS{Type: "name", Properties: map[string]string{"name": name}}
}

// Exaples:
//"crs": {
//  "type": "link",
//  "properties": {
//    "href": "http://example.com/crs/42",
//    "type": "proj4"
//    }
//  }
//
//"crs": {
//  "type": "link",
//  "properties": {
//    "href": "data.crs",
//    "type": "ogcwkt"
//    }
//  }
func NewLinkedCRS(href, typ string) *CRS {
	return &CRS{Type: "link",
		Properties: map[string]string{"href": href, "type": typ}}
}

func parseCoordinate(c interface{}) (coord Coordinate, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	coordinate, ok := c.([]interface{})
	if !ok || len(coordinate) != 2 {
		return Coordinate{}, fmt.Errorf("Error unmarshal %v to coordinates", c)
	}
	x := Coord(coordinate[0])
	y := Coord(coordinate[1])
	return Coordinate{x, y}, nil
}

func parseCoordinates(obj interface{}) (Coordinates, error) {
	c, ok := obj.([]interface{})
	if !ok || len(c) < 1 {
		return nil, fmt.Errorf("ParseErrr: Coordinates parse error, %v", obj)
	}
	coords := make(Coordinates, len(c))
	for i, v := range c {
		var err error
		if coords[i], err = parseCoordinate(v); err != nil {
			return nil, err
		}
	}
	return coords, nil
}

func parseMultiLine(obj interface{}) (MultiLine, error) {
	c, ok := obj.([]interface{})
	if !ok || len(c) < 1 {
		return nil, fmt.Errorf("ParseErrr: MultiLine parse error, %v", obj)
	}
	coords := make(MultiLine, len(c))
	for i, v := range c {
		var err error
		if coords[i], err = parseCoordinates(v); err != nil {
			return nil, err
		}
	}
	return coords, nil
}

func parsePoint(obj interface{}) (*Point, error) {
	c, err := parseCoordinate(obj)
	if err != nil {
		return nil, err
	}
	return NewPoint(c), nil
}

func parseLineString(obj interface{}) (*LineString, error) {
	cc, err := parseCoordinates(obj)
	if err != nil {
		return nil, err
	}
	return NewLineString(cc), nil
}

func parseMultiPoint(obj interface{}) (*MultiPoint, error) {
	cc, err := parseCoordinates(obj)
	if err != nil {
		return nil, err
	}
	return NewMultiPoint(cc), nil
}

func parseMultiLineString(obj interface{}) (*MultiLineString, error) {
	ml, err := parseMultiLine(obj)
	if err != nil {
		return nil, err
	}
	return NewMultiLineString(ml), nil
}

func parsePolygon(obj interface{}) (*Polygon, error) {
	pl, err := parseMultiLine(obj)
	if err != nil {
		return nil, err
	}
	return NewPolygon(pl), nil
}

func parseMultiPolygon(obj interface{}) (*MultiPolygon, error) {
	var ml []MultiLine
	si, ok := obj.([]interface{})
	if !ok {
		return nil, errors.New("Parse Error: parse multi polygon error")
	}
	ml = make([]MultiLine, len(si), len(si))

	for i, slice := range si {
		var err error
		if ml[i], err = parseMultiLine(slice); err != nil {
			return nil, err
		}
	}
	return NewMultiPolygon(ml), nil
}

func parseGeometry(gi interface{}) (Geometry, error) {
	g := gi.(map[string]interface{})
	coord := g["coordinates"]
	switch typ := g["type"]; typ {
	case "Point":
		return parsePoint(coord)
	case "LineString":
		return parseLineString(coord)
	case "MultiPoint":
		return parseMultiPoint(coord)
	case "MultiLineString":
		return parseMultiLineString(coord)
	case "Polygon":
		return parsePolygon(coord)
	case "MultiPolygon":
		return parseMultiPolygon(coord)
	case "GeometryCollection":
		return parseGeometryCollection(g["geometries"])
	default:
		return nil, fmt.Errorf("ParseError: Unknown geometry type %s", typ)
	}
}

func parseGeometryCollection(obj interface{}) (*GeometryCollection, error) {
	gc := NewGeometryCollection()
	si, ok := obj.([]interface{})
	if !ok {
		return nil, errors.New("ParseError: Error durring parse geometry collection")
	}
	for i := 0; i < len(si); i++ {
		geometry, err := parseGeometry(si[i])
		if err != nil {
			return nil, err
		}
		//gc.AddGeometry(si[i])
		gc.Geometries = append(gc.Geometries, geometry)
	}
	return gc, nil
}
