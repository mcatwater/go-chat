package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
)

type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	id string
}

type Message struct {
	Id      string
	Type    string
	To      string
	Message string
}

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var connections = make(map[string]*connection)

func login(u string, p string, ws *websocket.Conn) {

	db, err := sql.Open("mysql",
		"root@tcp(127.0.0.1:3306)/users")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	var (
		id       int
		username string
		password string
	)
	rows, err := db.Query("select * from users where username = ?", u)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&id, &username, &password)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(id, username, password)
		if p == password {
			log.Println("Login Successful")
			if _, ok := connections[u]; !ok {
				ws := connection{ws: ws, id: u}
				connections[u] = &ws
			}
			err = ws.WriteMessage(websocket.TextMessage, []byte("Login Successful"))
		} else {
			log.Println("Login Failed")
			err = ws.WriteMessage(websocket.TextMessage, []byte("Login Failed. Check your password or register"))
		}

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}

func register(u string, p string, ws *websocket.Conn) {

	db, err := sql.Open("mysql",
		"root@tcp(127.0.0.1:3306)/users")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	stmt, err := db.Prepare("insert into users (username, password) values (?,?);")
	if err != nil {
		log.Fatal(err)
	}
	res, err := stmt.Exec(u, p)
	if err != nil {
		log.Println("got here")
		log.Fatal(err)
	}
	lastId, err := res.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ID = %d, affected = %d\n", lastId, rowCnt)
	log.Println("Login Successful")
	if _, ok := connections[u]; !ok {
		ws := connection{ws: ws, id: u}
		connections[u] = &ws
	}
	err = ws.WriteMessage(websocket.TextMessage, []byte("Login Successful"))
}

func home(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		var m Message
		err = json.Unmarshal(message, &m)

		switch m.Type {
		case "login":
			log.Println("login")
			login(m.Id, m.Message, c)
		case "message":
			log.Println("message")

			if conn, ok := connections[m.To]; ok {
				err = conn.ws.WriteMessage(mt, []byte(m.Id+": "+m.Message))
			}
			if err != nil {
				log.Println("write:", err)
				break
			}
		case "logout":
			log.Println("logout")

		case "register":
			log.Println("register")
			register(m.Id, m.Message, c)

		}

	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	defer func() {
		for _, c := range connections {
			c.ws.Close()
		}
	}()

	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))

}
