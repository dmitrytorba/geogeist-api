package geogeist

import (
	"database/sql"
    "database/sql/driver"
    "time"
	"fmt"
	"log"
	"net/http"
    "net"
	"os"
    "golang.org/x/crypto/ssh"
    "golang.org/x/crypto/ssh/agent"
	"github.com/lib/pq"
)

type ViaSSHDialer struct {
    client *ssh.Client
}

func (self *ViaSSHDialer) Open(s string) (_ driver.Conn, err error) {
    return pq.DialOpen(self, s)
}

func (self *ViaSSHDialer) Dial(network, address string) (net.Conn, error) {
    return self.client.Dial(network, address)
}

func (self *ViaSSHDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
    return self.client.Dial(network, address)
}

var (
	db *sql.DB
    
    sshHost         = os.Getenv("dbip")
    sshPort         = 22
    sshUser         = os.Getenv("sshuser")
    sshPass         = os.Getenv("sshpass")
	connectionName  = os.Getenv("db")
	dbUser          = "geogeist"
	dbPassword      = os.Getenv("dbpass")
    sslMode         = os.Getenv("sslmode")
	dsn             = fmt.Sprintf("user=%s password=%s host=%s sslmode=%s", dbUser, dbPassword, connectionName, sslMode)
)

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

func init() {
    var agentClient agent.Agent
    // Establish a connection to the local ssh-agent
    if conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
        defer conn.Close()

        // Create a new instance of the ssh agent
        agentClient = agent.NewClient(conn)
    }

    // The client configuration with configuration option to use the ssh-agent
    sshConfig := &ssh.ClientConfig{
        User: sshUser,
        Auth: []ssh.AuthMethod{},
    }

    // When the agentClient connection succeeded, add them as AuthMethod
    if agentClient != nil {
        sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeysCallback(agentClient.Signers))
    }
    // When there's a non empty password add the password AuthMethod
    if sshPass != "" {
        sshConfig.Auth = append(sshConfig.Auth, ssh.PasswordCallback(func() (string, error) {
            return sshPass, nil
        }))
    }

    // Connect to the SSH Server
    if sshcon, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", sshHost, sshPort), sshConfig); err == nil {
        defer sshcon.Close()

        // Now we register the ViaSSHDialer with the ssh connection as a parameter
        sql.Register("postgres+ssh", &ViaSSHDialer{sshcon})

        // And now we can use our new driver with the regular postgres connection string tunneled through the SSH connection
        if db, err := sql.Open("postgres+ssh", dsn); err == nil {
            fmt.Printf("Successfully connected to the db\n")
            db.SetMaxIdleConns(1)
            db.SetMaxOpenConns(1)
        } else {
            fmt.Printf("Failed to connect to the db: %s\n", err.Error())
        }

    }
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
