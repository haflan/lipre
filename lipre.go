package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Not used - either read password from config / flag or define the valid rooms in a file on server
const temporaryCorrectRoomCode = "tester"

type File struct {
	Name     string `json:"name"`
	Contents string `json:"contents"`
}

type Room struct {
	mu        sync.Mutex
	code      string
	presenter *websocket.Conn
	viewers   []*websocket.Conn
	// Store files so that they can be sent to new viewers upon connection
	files map[string]File
	// Number of minutes for which the room should continue to be open after the presenter disconnects
	linger int
}

var rooms = make(map[string]*Room)

// Thread safe Room functions.
// The rooms map should only be updated from here
func (room *Room) Open() {
	room.mu.Lock()
	defer room.mu.Unlock()
	existingRoom := rooms[room.code]
	if existingRoom != nil {
		fmt.Printf("Closing existing room '%v'\n", existingRoom.code)
		existingRoom.Close(false)
	}
	rooms[room.code] = room
	room.presenter.SetCloseHandler(func(code int, text string) error {
		room.Close(true)
		return nil
	})
	go room.listen()
}

func (room *Room) Close(presenter bool) {
	room.mu.Lock()
	defer room.mu.Unlock()
	for _, viewer := range room.viewers {
		if viewer != nil {
			viewer.Close()
		}
	}
	// Close the presenter too, in case it's not closed already
	if !presenter {
		room.presenter.Close()
	}
	delete(rooms, room.code)
}

func (room *Room) listen() {
	for {
		var file File
		err := room.presenter.ReadJSON(&file)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			} else {
				log.Printf("Connection to room '%v' closed by presenter", room.code)
			}
			// TODO: Handle JSON errors (send info to presenter)
			return
		}
		room.files[file.Name] = file
		for _, viewerConn := range room.viewers {
			if viewerConn == nil {
				break
			}
			viewerConn.WriteJSON(&file)
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
	qparams := r.URL.Query()
	pLinger := qparams["linger"]
	var iLinger int
	if len(pLinger) == 1 {
		iLinger, _ = strconv.Atoi(pLinger[0])
	}
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
	room := &Room{code: roomCode, presenter: conn, linger: iLinger, files: make(map[string]File)}
	room.Open()
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
		conn.WriteJSON(&filedata)
	}
}

func main() {
	fmt.Println("Server starting")
	router := mux.NewRouter()
	router.HandleFunc("/ws/pres/{roomCode}", presentHandler)
	router.HandleFunc("/ws/view/{roomCode}", viewHandler)
	router.PathPrefix("/").HandlerFunc(fileHandler)
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", 8080), nil))
}
