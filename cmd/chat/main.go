package main

import (
    "time"
    "net/http"
    "log"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "strings"
)

type client struct {
    conn *websocket.Conn
    revision uint64
}

type messages struct {
    arr []string
    index int
    revision uint64
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
    r.revision++
}

func full_modulo(index int, length int) int {
    // -3 -2 -1 0 1 2 3 4 5 -> 0 -2 -1 0 1 2 0 1 2 -> 312012012 -> 012012012
    return ((index % length) + length) % length
}

func (r *messages) forEach(amount int, cb func(int, string)) {
    l := len(r.arr)
    if amount < 0 || amount > l { amount = l; }

    // Start from latest message minus amount
    start := full_modulo(r.index - amount, l)
    for _i := start; _i < start + amount; _i++ {
        i := full_modulo(_i, l)

        item := r.arr[i]
        if item == "" {
            break;
        }
        cb(i, item)
    }
}

var register chan *websocket.Conn
var unregister chan *client
var inbound chan string
var clients map[*client]bool

func main() {

    register = make(chan *websocket.Conn)
    unregister = make(chan *client)
    inbound = make(chan string, 20)
    clients = make(map[*client]bool)

    go aggregator()

    router := gin.Default()
    router.GET("/chat", chat)
    router.Run()
}

// TODO Force tick if revision delta >= length on inbound, to avoid message loss on mass inbound
func aggregator() {
    outboundTicker := time.NewTicker(500 * time.Millisecond)
    msgs := makeMessages(20)
    for {
        select {
        case m := <-inbound:
            msgs.add(m)
        case <-outboundTicker.C:
            for c := range clients {
                num_to_send := msgs.revision - c.revision;
                if num_to_send > 0 {
                    log.Printf("%s - rev=%d, curr=%d", c.conn.RemoteAddr().String(), msgs.revision, c.revision)
                    msgs.forEach(int(num_to_send), func (i int, s string) {
                        //log.Print("Message ", i)
                        c.conn.WriteMessage(websocket.TextMessage, []byte(s))
                    })
                    c.revision = msgs.revision
                }
            }
        case conn := <-register:
            c := client {
                conn: conn,
                revision: 0, // Get all messages on init
            }
            clients[&c] = true
            go client_loop(&c)
        case c := <-unregister:
            delete(clients, c)
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
    register <- conn
    c.Status(http.StatusOK);
}

func client_loop(c *client) {
    for {
        messageType, p, err := c.conn.ReadMessage()
        if err != nil {
            log.Print("Read message failed! ", err)
            unregister <- c
            return
        }
        if messageType == websocket.TextMessage {
            s := string(p)
            inbound <- s
        }
    }
}
