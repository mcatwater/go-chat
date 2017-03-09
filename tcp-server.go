package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"io"
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

var db *sql.DB

var key = []byte("a very very very very secret key") // 32 bytes

func login(u string, p string, ws *websocket.Conn) {

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

func send(m Message, c *websocket.Conn) error {
	if _, ok := connections[m.Id]; ok {
		if conn, ok := connections[m.To]; ok {
			err := conn.ws.WriteMessage(websocket.TextMessage, []byte(m.Id+": "+m.Message))
			ciphertext, err := encrypt(key, []byte(m.Message))
			if err != nil {
				log.Fatal(err)
			}
			stmt, err := db.Prepare("insert into logs (to_user, from_user, message) values (?,?,?);")
			if err != nil {
				log.Fatal(err)
			}
			res, err := stmt.Exec(m.Id, m.To, ciphertext)
			if err != nil {
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
			return err
		} else {
			c.WriteMessage(websocket.TextMessage, []byte("User is not logged in"))
		}
	} else {
		c.WriteMessage(websocket.TextMessage, []byte("Please login first"))
	}
	return nil
}

func home(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	for {
		_, message, err := c.ReadMessage()
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
			err = send(m, c)
			if err != nil {
				log.Println("write:", err)
				break
			}
		case "logout":
			log.Println("logout")
			if _, ok := connections[m.Id]; ok {
				connections[m.Id].ws.Close()
				delete(connections, m.Id)
			}
			log.Println("Logged out")

		case "register":
			log.Println("register")
			register(m.Id, m.Message, c)

		}

	}
}

func encrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

func decrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func main() {
	/*
		result, err := decrypt(key, ciphertext)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", result)
	*/
	flag.Parse()
	log.SetFlags(0)

	defer func() {
		for _, c := range connections {
			c.ws.Close()
		}
	}()

	db, err := sql.Open("mysql",
		"root@tcp(127.0.0.1:3306)/users")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))

}
