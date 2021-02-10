package main

import (
    "time"
    "net/http"
    "log"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "strings"
)

type messages struct {
    arr []string
    index int
}

func makeMessages(size int) messages {
    m := messages {
        arr: make([]string, size),
        index: 0,
    }
    return m
}

func (r *messages) add(s string) {
    r.arr[r.index] = s
    r.index = (r.index + 1) % len(r.arr)
}

func (r *messages) forEach(cb func(int, string)) {
    l := len(r.arr)
    for i := 0; i < l; i++ {
        index := r.index - 1 - i
        if index < 0 {
            index = l + (index % l);
        }
        item := r.arr[index]
        if item == "" {
            break;
        }
        cb(index, item)
    }
}

var inbound chan string
var broadcast chan string
var clients map[*websocket.Conn]bool

func main() {

    inbound = make(chan string)
    broadcast = make(chan string)
    clients = make(map[*websocket.Conn]bool)

    go aggregator()
    go broadcaster()

    router := gin.Default()
    router.GET("/chat", chat)
    router.Run()
}

func aggregator() {
    outboundTicker := time.NewTicker(2000 * time.Millisecond)
    msgs := makeMessages(20)
    for {
        select {
        case m := <-inbound:
            msgs.add(m)
        case <-outboundTicker.C:
            msgs.forEach(func (i int, s string) {
                broadcast <- s
            })
        }
    }
}

func broadcaster() {
    for {
        s := <-broadcast
        for c, _ := range clients {
            c.WriteMessage(websocket.TextMessage, []byte(s))
        }
    }
}

var upgrader = websocket.Upgrader {
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
    CheckOrigin: checkOrigin,
}

// WS no CORS
func checkOrigin(r *http.Request) bool {
    origin := r.Header.Get("origin")
    log.Print("Origin = ", origin)
    return strings.Index(origin, "http://localhost") == 0 || strings.Index(origin, "localhost") == 0
}

func chat(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Print("Chat init failed! ", err)
        return
    }
    go client(conn)
    c.Status(http.StatusOK);
}

func client(conn *websocket.Conn) {
    clients[conn] = true
    defer func() {
        clients[conn] = false
    }()

    for {
        messageType, p, err := conn.ReadMessage()
        if err != nil {
            log.Print("Read message failed! ", err)
            return
        }
        if messageType == websocket.TextMessage {
            s := string(p)
            inbound <- s
        }
    }
}
