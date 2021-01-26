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

func (room *Room) Close() {
	for _, viewer := range room.viewers {
		if viewer != nil {
			viewer.Close()
		}
	}
	delete(rooms, room.code)
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
			room.Close()
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

// fileHandler looks in ui/dist directory for static files matching the path
// Writes a 404 message if not found
func fileHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path
	if filePath == "/" {
		filePath = "/index.html"
	}
	// Check if the file exists among the static assets
	// At time of writing, this is only true for index.html and lipre.js,
	// but code splitting may be introduced and change that
	htmlData, err := ioutil.ReadFile(fmt.Sprintf("ui/dist%v", filePath))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 :("))
		return
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
	conn.SetCloseHandler(func(code int, text string) error {
		room.Close()
		return nil
	})
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
	router.HandleFunc("/ws/pres/{roomCode}", presentHandler)
	router.HandleFunc("/ws/view/{roomCode}", viewHandler)
	router.PathPrefix("/").HandlerFunc(fileHandler)
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", 8088), nil))
}
