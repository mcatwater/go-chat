package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var addr_client = flag.String("addr_client", "localhost:8000", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func echo(w http.ResponseWriter, r *http.Request) {
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

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

func home(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("client.html")
	t.Execute(w, "ws://"+*addr+"/echo")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr_client, nil))
	/*
		flag.Parse()
		log.SetFlags(0)

		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)

		u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
		log.Printf("connecting to %s", u.String())

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Fatal("dial:", err)
		}
		defer c.Close()

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

		for {
			select {
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
			default:
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Text to send: ")
				text, _ := reader.ReadString('\n')
				err := c.WriteMessage(websocket.TextMessage, []byte(text))
				if err != nil {
					log.Println("write:", err)
					return
				}
			}
		}
	*/
}
