package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	//go:embed ui/dist/index.html
	indexHTML []byte

	//go:embed ui/dist/lipre.js
	lipreJS []byte

	//go:embed ui/dist/favicon.ico
	favicon []byte

	//go:embed lipre.py
	liprePy string

	liprePyTemplate = template.Must(template.New("lipre.py").Parse(liprePy))
)

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
// The rooms map should only be written to from these

func (room *Room) open() {
	room.mu.Lock()
	defer room.mu.Unlock()
	existingRoom := rooms[room.code]
	if existingRoom != nil {
		existingRoom.close()
	}
	rooms[room.code] = room
	room.presenter.SetCloseHandler(func(code int, text string) error {
		room.close()
		return nil
	})
	go room.listen()
}

func (room *Room) close() {
	room.mu.Lock()
	defer room.mu.Unlock()
	// The actual room close code is handled in the presenter connection close handler
	fmt.Printf("Closing room '%v'\n", room.code)
	for _, viewer := range room.viewers {
		if viewer != nil {
			viewer.Close()
		}
	}
	room.presenter.Close()
	delete(rooms, room.code)
}

func (room *Room) addViewer(viewerConn *websocket.Conn) {
	room.mu.Lock()
	room.viewers = append(room.viewers, viewerConn)
	room.mu.Unlock()
	// And write all existing files
	for _, filedata := range room.files {
		viewerConn.WriteJSON(&filedata)
	}
}

func (room *Room) listen() {
	for {
		var file File
		err := room.presenter.ReadJSON(&file)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
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
	switch filePath {
	case "/":
		if strings.HasPrefix(r.Header.Get("User-Agent"), "curl") {
			goto liprepy
		}
		fallthrough
	case "/index.html":
		w.Write(indexHTML)
	case "/favicon.ico":
		w.Write(favicon)
	case "/lipre.js":
		w.Write(lipreJS)
	case "/lipre.py", "/l.py":
		goto liprepy
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 :("))
	}
	return
liprepy:
	// If this fails we get 0, which is the desired default anyway
	linger, _ := strconv.Atoi(r.URL.Query().Get("linger"))
	liprePyTemplate.Execute(w, struct {
		Host   string
		Linger int
	}{
		Host:   r.Host,
		Linger: linger,
	})
}

func presentHandler(w http.ResponseWriter, r *http.Request) {
	roomCode := mux.Vars(r)["roomCode"]
	qparams := r.URL.Query()
	pLinger := qparams["linger"]
	var iLinger int
	if len(pLinger) == 1 {
		iLinger, _ = strconv.Atoi(pLinger[0])
	}
	fmt.Println("Upgrading connection")
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	room := &Room{code: roomCode, presenter: conn, linger: iLinger, files: make(map[string]File)}
	room.open()
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
	room.addViewer(conn)
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
