package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var addr_client = flag.String("addr_client", "localhost:8001", "http service address")

func home(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("client.html")
	t.Execute(w, "ws://"+*addr+"/echo")
}

func main() {
	/*
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Port: ")
		port, _ := reader.ReadString('\n')
		if len(port) < 0 {
			port = "8000"
		}

		url := "localhost:" + port
		log.Println(url)
		addr_client := flag.String("addr_client", url, "http service address")
	*/
	flag.Parse()
	log.SetFlags(0)

	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr_client, nil))
}
