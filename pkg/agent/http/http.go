package http

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/lastbackend/lastbackend/pkg/agent/http/routes"
	"net/http"
	"strconv"
	"time"
	"github.com/Sirupsen/logrus"
)


type Handler struct {
	Path    string
	Method  string
	Auth    bool
	Handler func(http.ResponseWriter, *http.Request)
}

func NewRouter() *mux.Router {

	r := mux.NewRouter()
	r.Methods("OPTIONS").HandlerFunc(headers)

	// Session handlers
	r.HandleFunc("/system", handle(routes.VersionGetR)).Methods(http.MethodPost)

	// User handlers

	return r
}

func RunHttpServer(routes *mux.Router, port int) {
	logrus.Infof("Listen http server on %d port", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), routes); err != nil {
		logrus.Fatal("ListenAndServe: ", err)
	}
}

func headers(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	w.Header().Add("Access-Control-Allow-Origin", origin)
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Methods", "OPTIONS,GET,POST,PUT,DELETE")
	w.Header().Add("Access-Control-Allow-Headers", "X-CSRF-Token, Authorization, Content-Type, x-lastbackend, Origin, X-Requested-With, Content-Name, Accept")
	w.Header().Add("Content-Type", "application/json")
}

func handle(h http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	headers := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			headers(w, r)
			h.ServeHTTP(w, r)

			fmt.Println(fmt.Sprintf("%s\t%s\t%s", r.Method, r.RequestURI, time.Since(start)))
		}
	}

	h = headers(h)
	for _, m := range middleware {
		h = m(h)
	}

	return h
}