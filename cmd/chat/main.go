package main

import (
    "net/http"
    "log"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
	"github.com/starqi/wi-util-servers/cmd/chat/chat"
	"github.com/starqi/wi-util-servers/cmd/chat/sessions"
)

var sessionTokenHeader = "X-sessionToken"

var chatService *chat.Chat
var sessionsService *sessions.Sessions

func main() {

    sessionsService = sessions.MakeSessions()
    chatService = chat.MakeChat()

    router := gin.Default()
    router.GET("/chat", chatWs)
    // Should be rate limited by Nginx
    router.GET("/new_token", newToken)

    // Private via Nginx
    router.GET("/token/:id", describeToken)
    router.PATCH("/token/:id", patchToken)

    router.Run()
}

var upgrader = websocket.Upgrader {
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
    CheckOrigin: checkOrigin,
}

// WS has no CORS b/c of 101 protocol switch
// Will use session token validation   
func checkOrigin(r *http.Request) bool {
    return true
}

func chatWs(c *gin.Context) {
    st := c.Request.Header.Get(sessionTokenHeader)
    if st == "" {
        log.Print("Missing token header in chat connection ", c.ClientIP())
        c.Status(http.StatusUnauthorized)
        return
    }

    cb := make(chan *sessions.Session)
    defer close(cb)
    sessionsService.FindChan<-sessions.FindData{st, cb}
    session := <-cb

    if session == nil {
        log.Print("Invalid session in chat connection ", st, " ", c.ClientIP())
        c.Status(http.StatusUnauthorized)
        return
    }
    if !session.GetIsInGame() {
        log.Print("Not in-game for chat connection ", st, " ", c.ClientIP())
        c.Status(http.StatusUnauthorized)
        return
    }

    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Print("Chat init failed! ", err)
        return
    }
    chatService.Register <- conn
    c.Status(http.StatusOK);
}

func newToken(c *gin.Context) {
    cb := make(chan string)
    defer close(cb)
    sessionsService.RequestChan<-sessions.RequestData{cb}
    token := <-cb
    c.JSON(http.StatusOK, gin.H{"token": token})
}

func describeToken(c *gin.Context) {
    id, success := c.Params.Get("id")
    if !success {
        c.AbortWithStatus(http.StatusBadRequest)
        return
    }

    cb := make(chan *sessions.Session)
    defer close(cb)
    sessionsService.FindChan<-sessions.FindData{id, cb}

    if session := <-cb; session != nil {
        c.JSON(http.StatusOK, sessions.SessionToJson(session))
    } else {
        c.JSON(http.StatusOK, nil)
    }
}

func patchToken(c *gin.Context) {
    id, success := c.Params.Get("id")
    if !success {
        c.AbortWithStatus(http.StatusBadRequest)
        return
    }

    var json sessions.SessionAsJson
    if err := c.BindJSON(&json); err != nil {
        log.Print("Patch token JSON parse failed ", err)
        c.AbortWithStatus(http.StatusBadRequest)
        return
    }

    cb := make(chan bool);
    sessionsService.PatchFromJsonChan<-sessions.PatchFromJsonData{id, &json, cb}
    if success := <-cb; !success {
        log.Print("Patch token missing token ", json.Token)
        c.AbortWithStatus(http.StatusBadRequest)
        return
    }

    c.Status(http.StatusOK)
}
