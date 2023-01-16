package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const dbfile = "db/instausers.sqlite3"

var DB *sql.DB
var ProxyServers []string
var conNum = 10

type UserData struct {
	Data struct {
		User struct {
			Biography  string `json:"biography"`
			FullName   string `json:"full_name"`
			Username   string `json:"username"`
			ID         string `json:"id"`
			ProfilePic string `json:"profile_pic_url_hd"`
			IsPrivate  bool   `json:"is_private"`

			EdgeRelatedProfiles struct {
				Edges []struct {
					Node struct {
						FullName   string `json:"full_name"`
						Username   string `json:"username"`
						ID         string `json:"id"`
						IsPrivate  bool   `json:"is_private"`
						ProfilePic string `json:"profile_pic_url"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_related_profiles"`
		} `json:"user"`
	} `json:"data"`
}

func chkerr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func makereq(requrl string, proxyserver string) []byte {
	time.Sleep(time.Second * 1)

	proxyaddr, _ := url.Parse(proxyserver)
	httpTransport := &http.Transport{Proxy: http.ProxyURL(proxyaddr)}
	client := &http.Client{Transport: httpTransport, Timeout: time.Second * 10}

	req, err := http.NewRequest("GET", requrl, nil)
	chkerr(err)
	req.Header.Add("User-Agent", "Instagram 219.0.0.12.117 Android")
	// req.Header.Add("X-IG-App-ID", "936619743392459")

	resp, err := client.Do(req)
	if err != nil {
		return []byte("")
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte("")
	}
	return data
}

func saveUserRelData(userdata UserData) {
	insertUsrDat := `INSERT OR IGNORE INTO usersq VALUES(?);`
	user := userdata.Data.User

	for _, edge := range user.EdgeRelatedProfiles.Edges {
		reluser := edge.Node
		row := DB.QueryRow("SELECT username FROM userdata WHERE username = ?;", reluser.Username)
		var tmp string
		err := row.Scan(tmp)
		if err == sql.ErrNoRows {
			_, err := DB.Exec(insertUsrDat, reluser.Username)
			chkerr(err)
		}
	}
}

func saveUser(userdata UserData) {
	insertUsrDat := `INSERT INTO userdata VALUES(?,?,?,?,?)`
	user := userdata.Data.User
	if user.Username == "" {
		return
	}
	_, err := DB.Exec(insertUsrDat, user.Username, user.ID, user.Biography,
		user.ProfilePic, user.IsPrivate)
	chkerr(err)

	saveUserRelData(userdata)
}

func nextUserq() string {
	query := "select username from usersq where username not in (select username from userstmp) limit 1 ;"
	row := DB.QueryRow(query, nil)
	var username string
	err := row.Scan(&username)
	chkerr(err)
	ins := "insert into userstmp values(?);"
	_, err = DB.Exec(ins, username)
	chkerr(err)
	return username
}

func delUserQ(username string) {
	delquery := "delete from usersq where username = ?;"
	_, err := DB.Exec(delquery, username)
	chkerr(err)
	delquery = "delete from userstmp where username = ?;"
	_, err = DB.Exec(delquery, username)
	chkerr(err)
}

func initdb() {
	var err error
	DB, err = sql.Open("sqlite3", dbfile)
	chkerr(err)
}

func getUserData(username string, proxyserver string) (UserData, error) {
	requel := "https://www.instagram.com/api/v1/users/web_profile_info/?username=" + username
	data := makereq(requel, proxyserver)
	var userdata UserData
	err := json.Unmarshal(data, &userdata)
	return userdata, err
}

func proxyGet(id int, proxyservers []string, username <-chan string, userdata chan<- UserData) {
	i := 0
	for {
		uname := <-username
		for {
			proxyserver := proxyservers[i]
			log.Printf("func[%d]: user %s, proxy %s", id, uname, proxyserver)
			udat, err := getUserData(uname, proxyserver)
			if err == nil {
				userdata <- udat
				break
			} else {
				i = (i + 1) % len(proxyservers)
			}

		}
	}
}

func getinsta() {
	username := make(chan string)
	userdat := make(chan UserData)
	i := 0
	bracket := len(ProxyServers) / conNum
	for ; i < conNum-1; i++ {
		unm := nextUserq()
		go proxyGet(i, ProxyServers[i*bracket:(i+1)*bracket-1], username, userdat)
		username <- unm
	}
	unm := nextUserq()
	go proxyGet(i, ProxyServers[i*bracket:(i+1)*bracket-1], username, userdat)
	username <- unm

	for {
		userdata := <-userdat
		saveUser(userdata)
		delUserQ(userdata.Data.User.Username)
		log.Println("got user " + userdata.Data.User.Username)
		unm := nextUserq()
		username <- unm
	}

}

func LoadProxyServers() []string {
	f, err := os.ReadFile("proxy.txt")
	chkerr(err)
	return strings.Split(string(f), "\n")
}

func main() {
	log.Println("started")
	initdb()
	ProxyServers = LoadProxyServers()
	getinsta()
}
