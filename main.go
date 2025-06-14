package main

import (
    "context"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/Southclaws/go-samp-query"
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
        http.Error(w, "Missing host param (e.g., host=1.2.3.4:7777)", 400)
        return
    }

    ctx := context.Background()
    q, err := sampquery.NewLegacyQuery(host)
    if err != nil {
        http.Error(w, "Failed to connect: "+err.Error(), 500)
        return
    }
    defer q.Close()

    data, err := q.SendQuery(ctx, sampquery.QueryType(0x64)) // Opcode 'd'
    if err != nil {
        http.Error(w, "Query failed: "+err.Error(), 500)
        return
    }

    players := parsePlayers(data)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(players)
}

func parsePlayers(b []byte) []Player {
    // Skip initial header
    offset := 11
    count := int(b[offset])
    offset++

    players := make([]Player, 0, count)
    for i := 0; i < count; i++ {
        id := int(b[offset]); offset++

        nl := int(b[offset]); offset++
        name := string(b[offset : offset+nl])
        offset += nl

        score := int(binary.LittleEndian.Uint32(b[offset : offset+4]))
        offset += 4

        // Skip ping field (4 bytes), not needed
        offset += 4

        players = append(players, Player{
            ID:    id,
            Name:  name,
            Score: score,
        })
    }
    return players
}
