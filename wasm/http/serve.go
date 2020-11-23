package main

import (
	"log"
	"net/http"
	"path/filepath"
)

// httpConn web server connection parameters
type httpConn struct {
	address    string
	port       string
	root       string
	cascadeDir string
}

func main() {
	httpConn := &httpConn{
		address:    "localhost",
		port:       "5000",
		root:       "./",
		cascadeDir: "../cascade/",
	}
	c := NewConn(httpConn)
	c.Init()
}

// NewConn establish a new http connection
func NewConn(conn *httpConn) *httpConn {
	return conn
}

// Init listen and serves the connection endpoints
func (c *httpConn) Init() {
	var err error
	c.root, err = filepath.Abs(c.root)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("serving %s on %s:%s", c.root, c.address, c.port)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(c.root))))
	http.Handle("/cascade/", http.StripPrefix("/cascade/", http.FileServer(http.Dir(c.cascadeDir))))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print(r.RemoteAddr + " " + r.Method + " " + r.URL.String())
		http.DefaultServeMux.ServeHTTP(w, r)
	})
	httpServer := http.Server{
		Addr:    c.address + ":" + c.port,
		Handler: handler,
	}
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}
