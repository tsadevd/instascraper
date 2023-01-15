package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const dbfile = "db/instausers.sqlite3"

var DB *sql.DB

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

func makereq(requrl string) []byte {
	time.Sleep(time.Second * 5)
	client := &http.Client{}

	req, err := http.NewRequest("GET", requrl, nil)
	chkerr(err)
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:106.0) Gecko/20100101 Firefox/106.0")
	req.Header.Add("X-IG-App-ID", "936619743392459")

	resp, err := client.Do(req)
	chkerr(err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	return data
}

func saveUserRelData(userdata UserData) {
	insertUsrDat := `INSERT OR IGNORE INTO usersq VALUES(?)`
	user := userdata.Data.User

	for _, edge := range user.EdgeRelatedProfiles.Edges {
		reluser := edge.Node
		_, err := DB.Exec(insertUsrDat, reluser.Username)
		chkerr(err)
	}
}

func saveUser(userdata UserData) {
	insertUsrDat := `INSERT INTO userdata VALUES(?,?,?,?,?)`
	user := userdata.Data.User
	_, err := DB.Exec(insertUsrDat, user.Username, user.ID, user.Biography,
		user.ProfilePic, user.IsPrivate)
	chkerr(err)

	saveUserRelData(userdata)
}

func nextUserq() string {
	query := "select username from usersq limit 1;"
	row := DB.QueryRow(query, nil)
	var username string
	err := row.Scan(&username)
	chkerr(err)
	return username
}

func delUserQ(username string) {
	delquery := "delete from usersq where username = ?;"
	_, err := DB.Exec(delquery, username)
	chkerr(err)
}

func initdb() {
	var err error
	DB, err = sql.Open("sqlite3", dbfile)
	chkerr(err)
}

func getUserData(username string) UserData {
	requel := "https://www.instagram.com/api/v1/users/web_profile_info/?username=" + username
	data := makereq(requel)
	var userdata UserData
	err := json.Unmarshal(data, &userdata)
	chkerr(err)
	return userdata
}

func getinsta() {
	for {
		username := nextUserq()
		log.Println(username)
		userdata := getUserData(username)
		saveUser(userdata)
		delUserQ(username)
	}
}

func main() {
	log.Println("started")
	initdb()
	getinsta()
}
