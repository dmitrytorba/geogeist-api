package main

import (
//	"encoding/json"
    "log"
    //"net/http"
    //"github.com/gorilla/mux"
 	"database/sql"
    "fmt"
     _ "github.com/lib/pq"
)
 
const (
    DB_USER     = "geogeist"
    DB_PASSWORD = "password"
    DB_NAME     = "geogeist"
)
 
func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

func main() {
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
            DB_USER, DB_PASSWORD, DB_NAME)
    db, err := sql.Open("postgres", dbinfo)
    checkErr(err)
    defer db.Close()

    //lat := "38.6950877"
    //lon := "-121.2273314"
    row := db.QueryRow("SELECT c.name FROM counties c WHERE ST_Covers(c.geog, 'SRID=4326;POINT(-121.2273314 38.6950877)'::geography)")
    checkErr(err)
    var name string
    err = row.Scan(&name)
    checkErr(err)
    log.Println(name)

    //router := mux.NewRouter()
    //log.Fatal(http.ListenAndServe(":8000", router))
}