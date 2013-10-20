package main

import (
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

func setupDb(dbName, user, host string) (db *sql.DB) {
	conn := fmt.Sprintf("user=%s dbname=%s host=%s", dbName, user, host)
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

func GetTileFeatures(table string, bbox Envelope, pxbuf float64) (wkb [][]byte, err error) {
	// To reduce edge effects, buffer the geographic bounding box by the
	// amount of the largest (transformed) pixel stroke width for rendering, then
	// clip the resulting image back to the desired size
	buf := (bbox.W() / w) * pxbuf

	b := fmt.Sprintf("ST_Buffer(ST_MakeEnvelope(%f,%f,%f,%f, 3857), %f)",
		bbox.Min.X, bbox.Min.Y, bbox.Max.X, bbox.Max.Y, buf)

	tmpl := `SELECT ST_AsBinary(ST_Intersection(geom, %s)) 
            FROM %s where ST_Intersects(geom, %s);`
	sql := fmt.Sprintf(tmpl, b, table, b)
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

func TileRequestHandler(c *Config) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, rq *http.Request) {
		vars := mux.Vars(rq)

		x, _ := strconv.Atoi(vars["x"])
		y, _ := strconv.Atoi(vars["y"])
		z, _ := strconv.Atoi(vars["z"])
		bbox := TileToBbox(x, y, z)
		table := vars["table"]
		img, err := RenderTile(bbox, c.Layer[table])

		if err != nil {
			handleError(err, rw, "Bad request", 500)
		}
		writeImage(rw, img)
	}
}

func handleError(err error, w http.ResponseWriter, desc string, code int) {
	fmt.Println(err)
	http.Error(w, desc, code)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	conf, err := ParseConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	DbConn = setupDb(conf.Database.User, conf.Database.Name, conf.Database.Host)
	defer DbConn.Close()

	r := mux.NewRouter()
	r.HandleFunc("/tile/{table}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/",
		TileRequestHandler(conf))
	http.Handle("/", r)

	p := strconv.Itoa(*port)
	if err := http.ListenAndServe(":"+p, nil); err != nil {
		fmt.Println("Failed to start server: %v", err)
	} else {
		fmt.Println("Serving on port: " + p)
	}
}
