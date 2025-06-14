package main

import (
    "context"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "net/http"

    sampquery "github.com/Southclaws/go-samp-query"
)

type Player struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Score int    `json:"score"`
}

func main() {
    http.HandleFunc("/api/players", playersHandler)
    fmt.Println("Listening on :3000")
    http.ListenAndServe(":3000", nil)
}

func playersHandler(w http.ResponseWriter, r *http.Request) {
    host := r.URL.Query().Get("host")
    if host == "" {
        http.Error(w, "Missing host param (e.g., host=1.2.3.4:7777)", http.StatusBadRequest)
        return
    }

    ctx := context.Background()
    q, err := sampquery.New(host)
    if err != nil {
        http.Error(w, "Failed to connect: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer q.Close()

    data, err := q.SendQuery(ctx, sampquery.QueryType('d')) // 'd' = detailed player info
    if err != nil {
        http.Error(w, "Query failed: "+err.Error(), http.StatusInternalServerError)
        return
    }

    players := parsePlayers(data)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(players)
}

func parsePlayers(b []byte) []Player {
    offset := 11
    if len(b) < offset+1 {
        return nil
    }

    count := int(b[offset])
    offset++

    players := make([]Player, 0, count)
    for i := 0; i < count && offset < len(b); i++ {
        if offset+1 >= len(b) {
            break
        }

        id := int(b[offset])
        offset++

        nameLen := int(b[offset])
        offset++

        if offset+nameLen+8 > len(b) {
            break
        }

        name := string(b[offset : offset+nameLen])
        offset += nameLen

        score := int(binary.LittleEndian.Uint32(b[offset : offset+4]))
        offset += 4

        offset += 4 // skip ping

        players = append(players, Player{
            ID:    id,
            Name:  name,
            Score: score,
        })
    }
    return players
}
