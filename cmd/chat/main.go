package main

import (
    "net/http"
    "log"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
	"github.com/starqi/wi-util-servers/cmd/chat/chat"
	"github.com/starqi/wi-util-servers/cmd/chat/sessions"
)

var chatService *chat.Chat
var sessionsService *sessions.Sessions

func main() {

    sessionsService = sessions.MakeSessions()
    chatService = chat.MakeChat(sessionsService)

    // TODO CORS is for ease of local testing not behind Nginx, or else Chrome blocks requests to different ports
    router := gin.Default()
    router.Use(func (c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "*")
        c.Header("Access-Control-Allow-Headers", "Authorization, *")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
        } else {
            c.Next()
        }
    })
    router.GET("/chat", chatWs)
    // Should be rate limited by Nginx
    router.POST("/token/new", newToken)

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
    sessionsService.RequestChan<-sessions.RequestData{Cb: cb}
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
    sessionsService.FindChan<-sessions.FindData{Token: id, Cb: cb}

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
    sessionsService.PatchFromJsonChan<-sessions.PatchFromJsonData{Token: id, Info: &json, Cb: cb}
    if success := <-cb; !success {
        log.Print("Patch token missing token ", json.Token)
        c.AbortWithStatus(http.StatusBadRequest)
        return
    }

    c.Status(http.StatusOK)
}
