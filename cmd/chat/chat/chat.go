package chat

import (
    "time"
    "log"
    "github.com/gorilla/websocket"
)

type Chat struct {
    Register chan *websocket.Conn
    unregister chan *client
    inbound chan string
    clients map[*client]bool
}

func MakeChat() *Chat {
    chat := Chat {
        make(chan *websocket.Conn),
        make(chan *client),
        make(chan string, 20),
        make(map[*client]bool),
    }
    go chat.aggregator()
    return &chat
}

//////////////////////////////////////////////////

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

func fullModulo(index int, length int) int {
    // -3 -2 -1 0 1 2 3 4 5 -> 0 -2 -1 0 1 2 0 1 2 -> 312012012 -> 012012012
    return ((index % length) + length) % length
}

func (r *messages) forEach(amount int, cb func(int, string)) {
    l := len(r.arr)
    if amount < 0 || amount > l { amount = l; }

    // Start from latest message minus amount
    start := fullModulo(r.index - amount, l)
    for _i := start; _i < start + amount; _i++ {
        i := fullModulo(_i, l)

        item := r.arr[i]
        if item == "" {
            break;
        }
        cb(i, item)
    }
}

// TODO Force tick if revision delta >= length on inbound, to avoid message loss on mass inbound
func (chat *Chat) aggregator() {
    outboundTicker := time.NewTicker(500 * time.Millisecond)
    msgs := makeMessages(20)
    for {
        select {
        case m := <-chat.inbound:
            msgs.add(m)
        case <-outboundTicker.C:
            for c := range chat.clients {
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
        case conn := <-chat.Register:
            c := client {
                conn: conn,
                revision: 0, // Get all messages on init
            }
            chat.clients[&c] = true
            go chat.clientLoop(&c)
        case c := <-chat.unregister:
            delete(chat.clients, c)
        }
    }
}

func (chat *Chat) clientLoop(c *client) {
    for {
        messageType, p, err := c.conn.ReadMessage()
        if err != nil {
            log.Print("Closing client ", err) // Could just be disconnect
            chat.unregister <- c
            return
        }
        if messageType == websocket.TextMessage {
            s := string(p)
            chat.inbound <- s
        }
    }
}
