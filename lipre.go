package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Not used - either read password from config / flag or define the valid rooms in a file on server
const temporaryCorrectRoomCode = "tester"

type Room struct {
	code      string
	presenter *websocket.Conn
	viewers   []*websocket.Conn
	// Store files so that they can be sent to new viewers upon connection
	files map[string][]byte
}

// TODO: Lock on this for concurrency?
var rooms = make(map[string]*Room)

func (room *Room) listen() {
	for {
		_, message, err := room.presenter.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			delete(rooms, room.code)
			return
		}
		file := struct {
			Name     string `json:"name"`
			Contents string `json:"contents"`
		}{}
		err = json.Unmarshal(message, &file)
		if err != nil {
			log.Printf("error: %v", err)
			// TODO: Send info to presenter
			return
		}
		room.files[file.Name] = message
		for _, viewerConn := range room.viewers {
			if viewerConn == nil {
				break
			}
			viewerConn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

// HTTP
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	htmlData, err := ioutil.ReadFile("ui/index.html")
	if err != nil {
		panic(err)
	}
	w.Write(htmlData)
	return
}

func jsHandler(w http.ResponseWriter, r *http.Request) {
	htmlData, err := ioutil.ReadFile("ui/lipre.js")
	if err != nil {
		panic(err)
	}
	w.Write(htmlData)
	return
}

func presentHandler(w http.ResponseWriter, r *http.Request) {
	roomCode := mux.Vars(r)["roomCode"]
	/*if roomCode != temporaryCorrectRoomCode {
		w.WriteHeader(http.StatusBadRequest)
		return
	}*/
	fmt.Println("Upgrading connection")
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	room := &Room{code: roomCode, presenter: conn, files: make(map[string][]byte)}
	rooms[roomCode] = room
	go room.listen()
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	roomCode := mux.Vars(r)["roomCode"]
	room := rooms[roomCode]
	if room == nil {
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	room.viewers = append(rooms[roomCode].viewers, conn)
	for _, filedata := range room.files {
		conn.WriteMessage(websocket.TextMessage, filedata)
	}
}

func main() {
	fmt.Println("Server starting")
	router := mux.NewRouter()
	router.HandleFunc("/", indexHandler)
	router.HandleFunc("/room/{roomCode}", indexHandler)
	router.HandleFunc("/lipre.js", jsHandler) // Temporary solution?
	router.HandleFunc("/pres/{roomCode}", presentHandler)
	router.HandleFunc("/view/{roomCode}", viewHandler)
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", 8080), nil))
}
