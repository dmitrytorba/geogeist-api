package geogeist

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	_ "github.com/lib/pq"
)

var (
	db *sql.DB

	host = os.Getenv("db")
	pass = os.Getenv("dbpass")
	dsn = fmt.Sprintf("database=geogeist user=geogeist password=%s host=%s", pass, host)
)

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

func init() {
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Could not open db: %v", err)
	}

    log.Println("DB Connected")
	// Only allow 1 connection to the database to avoid overloading it.
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
}

func SetDb(db *sql.DB) {
    db = db
}

func GetLocation(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	lon := params["lon"][0]
	lat := params["lat"][0]

    // using $1 syntax for coords throws invalid geometry error
    // TODO figure out why
    coords := fmt.Sprintf("%s %s", lon, lat)
	log.Println(coords)
    row := db.QueryRow("SELECT c.state, c.data FROM states c WHERE ST_Covers(c.geog, 'SRID=4326;POINT(" + coords + ")'::geography)")
    var stateFips string
    var stateData string
    err := row.Scan(&stateFips, &stateData)
    checkErr(err)
    row = db.QueryRow("SELECT c.data, c.county FROM counties c WHERE c.state = $1 AND ST_Covers(c.geog, 'SRID=4326;POINT(" + coords + ")'::geography)", stateFips)
    var countyData string
    var county string
    err = row.Scan(&countyData, &county)
    checkErr(err)
    row = db.QueryRow("SELECT t.data FROM tracts t WHERE t.state = $1 AND t.county = $2 AND ST_Covers(t.geog, 'SRID=4326;POINT(" + coords + ")'::geography)", stateFips, county)
    var tractData string
    err = row.Scan(&tractData)
    checkErr(err)
    row = db.QueryRow("SELECT c.data FROM places c WHERE c.state = $1 AND ST_Covers(c.geog, 'SRID=4326;POINT(" + coords + ")'::geography)", stateFips)
    var placeData string
    err = row.Scan(&placeData)
    if err != sql.ErrNoRows {
        checkErr(err)
    } 
    s := fmt.Sprintf("{\"state\":%s,\"county\":%s,\"place\":%s,\"tract\":%s}", stateData, countyData, placeData, tractData)
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(s))
}
