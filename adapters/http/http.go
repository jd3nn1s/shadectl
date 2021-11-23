package http

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"net/url"
	"shadectl"
	"strconv"
	"time"
)

type PrimaryAdapter struct {
	router *mux.Router
	service shadectl.Service
	server *http.Server
}

func NewHTTPPrimaryAdapter(service shadectl.Service) {
	r := mux.NewRouter()

	h := PrimaryAdapter{
		router: r,
		service: service,
		server: &http.Server{
			Handler: r,
			Addr: "127.0.0.1:8000",
			WriteTimeout: 15 * time.Second,
			ReadTimeout: 15 * time.Second,
		},
	}

	r.HandleFunc("/window/up", h.up).Methods("POST")
	r.HandleFunc("/window/down", h.down).Methods("POST")
	r.HandleFunc("/window/position", h.position).Methods("GET")

	log.Fatal(h.server.ListenAndServe())
}

func (h *PrimaryAdapter) up(w http.ResponseWriter, r *http.Request) {
	// up and down and the same as we just rely on the position
	h.down(w, r)
}

func (h *PrimaryAdapter) down(w http.ResponseWriter, r *http.Request) {
	pos, err := extractPos(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.service.SetPosition(pos)
	if err != nil {
		http.Error(w, fmt.Sprintln("unable to set blind position", err), http.StatusBadRequest)
		return
	}
}

func (h *PrimaryAdapter) position(w http.ResponseWriter, r *http.Request) {
	pos, err := h.service.GetPosition()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write([]byte(strconv.Itoa(pos)))
	if err != nil {
		log.Println("unable to write position: ", err)
	}
	return
}

func extractPos(values url.Values) (int, error) {
	if pos, ok := values["pos"]; !ok {
		// respond with error saying pos must be present
		return -1, errors.New("request must have a 'pos' query parameter with a numeric value")
	} else {
		if len(pos) != 1 {
			return -1, errors.New("request must only have a single 'pos' query parameter")
		}
		return strconv.Atoi(pos[0])
	}
}