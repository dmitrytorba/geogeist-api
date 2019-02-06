package main

import (
//	"encoding/json"
    "log"
    "net/http"
    "github.com/gorilla/mux"
 	"database/sql"
    "fmt"
    _ "github.com/lib/pq"
)
 
const (
    DB_USER     = "geogeist"
    DB_PASSWORD = "password"
    DB_NAME     = "geogeist"
)

var db *sql.DB
 
func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

func connectDb() {
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
            DB_USER, DB_PASSWORD, DB_NAME)
	var err error
    db, err = sql.Open("postgres", dbinfo)
    checkErr(err)
}

func GetLocation(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	lon := params["lon"]
	lat := params["lat"]
	log.Println(lon)
	log.Println(lat)

    // using $1 syntax throws invalid geometry error
    // TODO figure out why
    coords := "-121.2273314 38.6950877"
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
    w.Write([]byte(s))
}

func main() {
	connectDb()
    defer db.Close()
    router := mux.NewRouter()
    router.HandleFunc("/coords/{lon}/{lat}", GetLocation).Methods("GET")
    const port = ":8000"
    log.Println("Serving on " + port)
    log.Fatal(http.ListenAndServe(port, router))
}