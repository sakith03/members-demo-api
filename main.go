package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/mail"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Member struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type memberInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Each process has its own demo data; restarting restores these two members.
type api struct {
	mu      sync.RWMutex
	members map[int]Member
	nextID  int
}

func newAPI() http.Handler {
	a := &api{
		members: map[int]Member{
			1: {ID: 1, Name: "Alice Silva", Email: "alice@example.com", Role: "admin"},
			2: {ID: 2, Name: "Bob Perera", Email: "bob@example.com", Role: "member"},
		},
		nextID: 3,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"service":   "members-demo-api",
			"endpoints": []string{"GET /healthz", "GET /members", "POST /members", "GET /members/{id}", "PUT /members/{id}", "DELETE /members/{id}"},
		})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "members-demo-api"})
	})
	mux.HandleFunc("GET /members", a.list)
	mux.HandleFunc("POST /members", a.create)
	mux.HandleFunc("GET /members/{id}", a.get)
	mux.HandleFunc("PUT /members/{id}", a.update)
	mux.HandleFunc("DELETE /members/{id}", a.delete)
	return mux
}

func (a *api) list(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	members := make([]Member, 0, len(a.members))
	for _, member := range a.members {
		members = append(members, member)
	}
	a.mu.RUnlock()
	sort.Slice(members, func(i, j int) bool { return members[i].ID < members[j].ID })
	writeJSON(w, http.StatusOK, map[string]any{"data": members, "count": len(members)})
}

func (a *api) create(w http.ResponseWriter, r *http.Request) {
	input, ok := readInput(w, r)
	if !ok {
		return
	}
	a.mu.Lock()
	member := Member{ID: a.nextID, Name: input.Name, Email: input.Email, Role: input.Role}
	a.members[member.ID] = member
	a.nextID++
	a.mu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]any{"data": member})
}

func (a *api) get(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	a.mu.RLock()
	member, exists := a.members[id]
	a.mu.RUnlock()
	if !exists {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": member})
}

func (a *api) update(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	input, ok := readInput(w, r)
	if !ok {
		return
	}
	a.mu.Lock()
	_, exists := a.members[id]
	member := Member{ID: id, Name: input.Name, Email: input.Email, Role: input.Role}
	if exists {
		a.members[id] = member
	}
	a.mu.Unlock()
	if !exists {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": member})
}

func (a *api) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	a.mu.Lock()
	_, exists := a.members[id]
	delete(a.members, id)
	a.mu.Unlock()
	if !exists {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "member id must be a positive integer")
		return 0, false
	}
	return id, true
}

func readInput(w http.ResponseWriter, r *http.Request) (memberInput, bool) {
	var input memberInput
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "use Content-Type: application/json")
		return input, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeInputError(w, err)
		return input, false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		writeInputError(w, err)
		return input, false
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Role = strings.TrimSpace(input.Role)
	if input.Name == "" || input.Email == "" {
		writeError(w, http.StatusBadRequest, "name and email are required")
		return input, false
	}
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email {
		writeError(w, http.StatusBadRequest, "email must be a valid email address")
		return input, false
	}
	if input.Role == "" {
		input.Role = "member"
	}
	return input, true
}

func writeInputError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "JSON body must be at most 64 KiB")
		return
	}
	writeError(w, http.StatusBadRequest, "send one JSON object with name, email, and optional role")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newAPI(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Printf("Members Demo API listening on :%s\n", port)
	log.Fatal(server.ListenAndServe())
}
