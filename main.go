package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Player holds ID, name and score of a connected player
type Player struct {
	ID         int    `json:"id"`
	Playername string `json:"playername"`
	Score      int    `json:"score"`
}

// buildQueryPacket constructs the UDP query packet for SA-MP servers
func buildQueryPacket(ip string, port int, opcode byte) []byte {
	buf := new(bytes.Buffer)
	buf.WriteString("SAMP")

	// Write IP in 4 bytes
	for _, part := range strings.Split(ip, ".") {
		b, _ := strconv.Atoi(part)
		buf.WriteByte(byte(b))
	}

	// Write port (little-endian)
	buf.WriteByte(byte(port & 0xFF))
	buf.WriteByte(byte((port >> 8) & 0xFF))

	// Write opcode ('d' = player list)
	buf.WriteByte(opcode)

	return buf.Bytes()
}

// queryPlayers sends the player query packet and parses the response
func queryPlayers(ip string, port int) ([]Player, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)

	conn, err := net.DialTimeout("udp", addr, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %v", err)
	}
	defer conn.Close()

	packet := buildQueryPacket(ip, port, 'd')
	if _, err := conn.Write(packet); err != nil {
		return nil, fmt.Errorf("failed to send packet: %v", err)
	}

	// Read response
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	if n <= 11 {
		return nil, fmt.Errorf("invalid response size")
	}

	data := buf[11:n] // Remove 11-byte SAMP header
	reader := bytes.NewReader(data)

	var playerCount uint8
	if err := binary.Read(reader, binary.LittleEndian, &playerCount); err != nil {
		return nil, fmt.Errorf("failed to read player count: %v", err)
	}

	var players []Player
	for i := 0; i < int(playerCount); i++ {
		var nameLen uint8
		if err := binary.Read(reader, binary.LittleEndian, &nameLen); err != nil {
			return nil, fmt.Errorf("failed to read name length: %v", err)
		}

		nameBytes := make([]byte, nameLen)
		if _, err := reader.Read(nameBytes); err != nil {
			return nil, fmt.Errorf("failed to read player name: %v", err)
		}

		var score int32
		if err := binary.Read(reader, binary.LittleEndian, &score); err != nil {
			return nil, fmt.Errorf("failed to read score: %v", err)
		}

		players = append(players, Player{
			ID:         i,
			Playername: string(nameBytes),
			Score:      int(score),
		})
	}

	return players, nil
}

// handler serves the /api/players endpoint
func handler(w http.ResponseWriter, r *http.Request) {
	ipParam := r.URL.Query().Get("ip")
	if ipParam == "" || !strings.Contains(ipParam, ":") {
		http.Error(w, "Missing or invalid ?ip=IP:PORT parameter", http.StatusBadRequest)
		return
	}

	parts := strings.Split(ipParam, ":")
	ip := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil || port <= 0 || port > 65535 {
		http.Error(w, "Invalid port number", http.StatusBadRequest)
		return
	}

	players, err := queryPlayers(ip, port)
	if err != nil {
		http.Error(w, "Query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

func main() {
	http.HandleFunc("/api/players", handler)
	log.Println("✅ Listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
