package main

import (
	"bufio"
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
	"os"
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
	fmt.Println(x, y, z, size)
	fmt.Println(x * size)
	bbox.Min = Point{Origin.X + x*size, Origin.Y - (y+1)*size}
	bbox.Max = Point{Origin.X + (x+1)*size, Origin.Y - y*size}
	return
}

func GetTileFeatures(bbox Envelope) (wkb [][]byte, err error) {
	b := fmt.Sprintf("ST_MakeEnvelope(%f,%f,%f,%f, 3857)",
		bbox.Min.X, bbox.Min.Y, bbox.Max.X, bbox.Max.Y)
	tmpl := `SELECT ST_AsBinary(ST_Intersection(geom, %s)) 
            FROM routes where ST_Intersects(geom, %s);`
	sql := fmt.Sprintf(tmpl, b, b)
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

func saveToPngFile(filePath string, m image.Image) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	b := bufio.NewWriter(f)
	err = png.Encode(b, m)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = b.Flush()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s OK.\n", filePath)
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

	geoms, err := GetTileFeatures(bbox)
	if err != nil {
		fmt.Println(err)
		http.Error(res, "Bad Tile", 500)
		return
	}

	for _, wkb := range geoms {
		geom, err := geos.FromWKB(wkb)
		if err != nil {
			fmt.Println(err)
			http.Error(res, "Bad Tile", 500)
		}

		coords, _ := geom.Coords()
		for i, c := range coords {
			pt := GeoPToImgP(c, bbox)
			if i == 0 {
				gc.MoveTo(pt.X, pt.Y)
			} else {
				gc.LineTo(pt.X, pt.Y)
			}
		}
		gc.Stroke()
	}

	saveToPngFile("TestPath.png", i)
}

func main() {
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
	r.HandleFunc("/tile/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/", RenderTile)
	http.Handle("/", r)

	p := strconv.Itoa(*port)
	if err := http.ListenAndServe(":"+p, nil); err != nil {
		fmt.Println("Failed to start server: %v", err)
	} else {
		fmt.Println("Serving on port: " + p)
	}
}
