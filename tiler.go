package main

import (
	"code.google.com/p/draw2d/draw2d"
	"code.google.com/p/gcfg"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/bmizerany/pq"
	"github.com/gorilla/mux"
	"github.com/paulsmith/gogeos/geos"
	"image"
	"image/png"
	"log"
	"math"
	"net/http"
	"runtime"
	"strconv"
)

type Point struct {
	X, Y float64
}

type Envelope struct {
	Min, Max Point
}

func (e *Envelope) W() float64 {
	return e.Max.X - e.Min.X
}

func (e *Envelope) H() float64 {
	return e.Max.Y - e.Min.Y
}

type Config struct {
	Database struct {
		Name string
		User string
	}
}

const (
	w       = 256.0
	h       = 256.0
	mapSize = 20037508.34789244 * 2.0
)

var (
	Origin = Point{-20037508.34789244, 20037508.34789244}
	port   = flag.Int("port", 7979, "Server Port")
	DbConn *sql.DB
)

func setupDb(dbName, user string) (db *sql.DB) {
	conn := fmt.Sprintf("user=%s dbname=%s", dbName, user)
	fmt.Println(conn)
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Fatalf("Bad db conn: %v", err)
	}
	return
}

func TileToBbox(xc, yc, zoom int) (bbox Envelope) {
	x := float64(xc)
	y := float64(yc)
	z := float64(zoom)
	size := mapSize / math.Pow(2, z)

	bbox.Min = Point{Origin.X + x*size, Origin.Y - (y+1)*size}
	bbox.Max = Point{Origin.X + (x+1)*size, Origin.Y - y*size}
	return
}

func GetTileFeatures(table string, bbox Envelope) (wkb [][]byte, err error) {
	// TODO: Clean table string
	b := fmt.Sprintf("ST_MakeEnvelope(%f,%f,%f,%f, 3857)",
		bbox.Min.X, bbox.Min.Y, bbox.Max.X, bbox.Max.Y)
	tmpl := `SELECT ST_AsBinary(ST_Intersection(geom, %s)) 
            FROM %s where ST_Intersects(geom, %s);`
	sql := fmt.Sprintf(tmpl, b, table, b)
	fmt.Println(sql)
	s, err := DbConn.Prepare(sql)
	if err != nil {
		return
	}

	rs, err := s.Query()
	if err != nil {
		fmt.Println(err)
		return
	}

	var geom []byte
	for rs.Next() {
		err = rs.Scan(&geom)
		if err != nil {
			return
		}
		wkb = append(wkb, geom)
	}
	return
}

func GeoPToImgP(geoP geos.Coord, b Envelope) Point {
	left := b.Min.X
	top := b.Max.Y
	x := (geoP.X - left) / (b.W() / w)
	y := (top - geoP.Y) / (b.H() / h)
	return Point{x, y}
}

func writeImage(w http.ResponseWriter, i image.Image) {
	w.Header().Set("Content-type", "image/png")
	err := png.Encode(w, i)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Bad Image", 500)
	}
}

func RenderTile(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	i := image.NewRGBA(image.Rect(0, 0, w, h))
	gc := draw2d.NewGraphicContext(i)
	gc.SetLineWidth(1)

	x, _ := strconv.Atoi(vars["x"])
	y, _ := strconv.Atoi(vars["y"])
	z, _ := strconv.Atoi(vars["z"])
	bbox := TileToBbox(x, y, z)

	geoms, err := GetTileFeatures(vars["table"], bbox)
	if err != nil {
		fmt.Println(err)
		http.Error(res, "Bad Tile", 500)
		return
	}

	fmt.Println("Tile features:", len(geoms))
	for _, wkb := range geoms {
		geom, err := geos.FromWKB(wkb)
		if err != nil {
			handleError(err, res, "Bad Tile", 500)
			return
		}
		t, err := geom.Type()
		if err != nil {
			handleError(err, res, "Couldn't parse geom", 500)
			return
		}

		switch t {
		case geos.LINESTRING, geos.MULTILINESTRING:
			renderLine(gc, geom, bbox)
		case geos.POLYGON, geos.MULTIPOLYGON:
			renderPolygon(gc, geom, bbox)
		default:
			handleError(nil, res, fmt.Sprintf("Unkown Geom Type: %s", t), 500)
		}
	}

	writeImage(res, i)
}

func renderPolygon(gc *draw2d.ImageGraphicContext, geom *geos.Geometry, bbox Envelope) {
	// Does not handle holes
	shell, err := geom.Shell()
	fmt.Println(shell)
	if err != nil {
		renderLine(gc, geom, bbox)
		return
	}
	renderLine(gc, shell, bbox)
}

func renderLine(gc *draw2d.ImageGraphicContext, geom *geos.Geometry, bbox Envelope) {
	coords, _ := geom.Coords()
	for idx, c := range coords {
		pt := GeoPToImgP(c, bbox)
		if idx == 0 {
			gc.MoveTo(pt.X, pt.Y)
		} else {
			gc.LineTo(pt.X, pt.Y)
		}
	}
	gc.Stroke()
}

func handleError(err error, w http.ResponseWriter, desc string, code int) {
	fmt.Println(err)
	http.Error(w, "Bad geom type", 500)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	var conf Config
	err := gcfg.ReadFileInto(&conf, "settings.conf")
	if err != nil {
		fmt.Println("Invalid setting.conf file", err)
		return
	}

	DbConn = setupDb(conf.Database.User, conf.Database.Name)
	defer DbConn.Close()

	r := mux.NewRouter()
	r.HandleFunc("/tile/{table}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/", RenderTile)
	http.Handle("/", r)

	p := strconv.Itoa(*port)
	if err := http.ListenAndServe(":"+p, nil); err != nil {
		fmt.Println("Failed to start server: %v", err)
	} else {
		fmt.Println("Serving on port: " + p)
	}
}
