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

	connectionName = os.Getenv("db")
	dbUser         = "geogeist"
	dbPassword     = os.Getenv("dbpass")
    sslMode        = os.Getenv("sslmode")
	dsn            = fmt.Sprintf("user=%s password=%s host=%s sslmode=%s", dbUser, dbPassword, connectionName, sslMode)
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

    // using $1 syntax throws invalid geometry error
    // TODO figure out why
    coords := fmt.Sprintf("%s %s", lon, lat)
	log.Println(coords)
    row := db.QueryRow("SELECT c.state, c.data FROM states c WHERE ST_Covers(c.geog, 'SRID=4326;POINT(" + coords + ")'::geography)")
    var stateFips string
    var stateData string
    err := row.Scan(&stateFips, &stateData)
    checkErr(err)
    row = db.QueryRow("SELECT c.data FROM counties c WHERE c.state = $1 AND ST_Covers(c.geog, 'SRID=4326;POINT(" + coords + ")'::geography)", stateFips)
    var countyData string
    err = row.Scan(&countyData)
    checkErr(err)
    row = db.QueryRow("SELECT c.data FROM places c WHERE c.state = $1 AND ST_Covers(c.geog, 'SRID=4326;POINT(" + coords + ")'::geography)", stateFips)
    var placeData string
    err = row.Scan(&placeData)
    if err == sql.ErrNoRows {
    	w.Write([]byte(""))
    	return
    } 
    checkErr(err)
    s := fmt.Sprintf("{\"state\":%s,\"county\":%s,\"place\":%s}", stateData, countyData, placeData)
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(s))
}
