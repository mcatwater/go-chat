package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var addr_client = flag.String("addr_client", "localhost:8000", "http service address")

var count = 0

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
} // use default options

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func send(c *websocket.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		count++
		message := "test " + strconv.Itoa(count)
		err := c.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("write:", err)
		}
		fmt.Fprintf(w, "test %d", count)
	}
}

func echo(c *websocket.Conn) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		//	defer c.Close()

		//	done := make(chan struct{})
		/*
			go func() {
				defer c.Close()
				defer close(done)
				for {
					_, message, err := c.ReadMessage()
					if err != nil {
						log.Println("read:", err)
						return
					}
					log.Printf("recv: %s", message)
				}
			}()
		*/
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			log.Printf("recv: %s", message)
			//		err = c.WriteMessage(mt, message)
			err = c.WriteMessage(mt, []byte(strings.ToUpper(string(message))))
			if err != nil {
				log.Println("write:", err)
				break
			}
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("client.html")
	t.Execute(w, "ws://"+*addr+"/echo")
}

func ping_pong(c *websocket.Conn) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	go func() {
		defer c.Close()
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			log.Println(t.String())
			if c != nil {
				err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
				if err != nil {
					log.Println("write:", err)
					return
				}
			} else {
				log.Println("danger danger" + t.String())
			}

		case <-interrupt:
			log.Println("interrupt")
			// To cleanly close a connection, a client should send a close
			// frame and wait for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			c.Close()
			return
		}
	}

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

	err_err := c.WriteMessage(websocket.TextMessage, []byte("danger danger"))
	if err_err != nil {
		log.Fatal(err_err)
	}
	go ping_pong(c)

	defer c.Close()

	http.HandleFunc("/echo", echo(c))
	http.HandleFunc("/", home)
	http.HandleFunc("/send", send(c))
	log.Fatal(http.ListenAndServe(*addr_client, nil))

}
