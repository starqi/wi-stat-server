package sessions

import (
    "time"
)

type Session struct {
    Token string
    IsInGame bool
    GameInstance string
    PlayerName string
    Expiry time.Time
}

type SessionAsJson struct {
    Token string `json:"token"`
    GameInstance string `json:"gameInstance"`
    IsInGame bool `json:"isInGame"`
    PlayerName string `json:"playerName"`
}

type PatchSessionRequest struct {
    GameInstance *string `json:"gameInstance"`
    IsInGame *bool `json:"isInGame"`
    PlayerName *string `json:"playerName"`
}

type PatchFromJsonData struct{Token string; Info *PatchSessionRequest; Cb chan bool}
type FindData struct{Token string; Cb chan *Session}
type RequestData struct{Cb chan string}

type Sessions struct {
    tokens map[string]*Session
    PatchFromJsonChan chan PatchFromJsonData
    FindChan chan FindData
    RequestChan chan RequestData
}
