package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var addr_client = flag.String("addr_client", "localhost:8000", "http service address")

var count = 0

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
} // use default options

func send(c *websocket.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		log.Println(r.FormValue("msg"))
		message := r.FormValue("msg")
		err := c.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("write:", err)
		}
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
		}
		log.Printf("recv: %s", msg)

		fmt.Fprintf(w, message)
	}
}

func echo(c *websocket.Conn) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		for {
			log.Println("forever loop3")
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			log.Printf("recv: %s", message)
			err = c.WriteMessage(mt, []byte(strings.ToUpper(string(message))))
			if err != nil {
				log.Println("write:", err)
				break
			}
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("client2.html")
	t.Execute(w, "ws://"+*addr+"/echo")
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	defer c.Close()

	http.HandleFunc("/echo", echo(c))
	http.HandleFunc("/", home)
	http.HandleFunc("/send", send(c))
	log.Fatal(http.ListenAndServe(*addr_client, nil))
}
