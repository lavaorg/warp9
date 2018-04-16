// +build httpstats

package warp9

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

var mux sync.RWMutex
var stat map[string]http.Handler
var httponce sync.Once

func register(s string, h http.Handler) {
	mux.Lock()
	if stat == nil {
		stat = make(map[string]http.Handler)
	}

	if h == nil {
		delete(stat, s)
	} else {
		stat[s] = h
	}
	mux.Unlock()
}
func (srv *Srv) statsRegister() {
	httponce.Do(func() {
		http.HandleFunc("/go9p/", StatsHandler)
		go http.ListenAndServe(":6060", nil)
	})

	register("/go9p/srv/"+srv.Id, srv)
}

func (srv *Srv) statsUnregister() {
	register("/go9p/srv/"+srv.Id, nil)
}

func (srv *Srv) ServeHTTP(c http.ResponseWriter, r *http.Request) {
	io.WriteString(c, fmt.Sprintf("<html><body><h1>Server %s</h1>", srv.Id))
	defer io.WriteString(c, "</body></html>")

	// connections
	io.WriteString(c, "<h2>Connections</h2><p>")
	srv.Lock()
	defer srv.Unlock()
	if len(srv.conns) == 0 {
		io.WriteString(c, "none")
		return
	}

	for _, conn := range srv.conns {
		io.WriteString(c, fmt.Sprintf("<a href='/go9p/srv/%s/conn/%s'>%s</a><br>", srv.Id, conn.Id, conn.Id))
	}
}

func (conn *Conn) statsRegister() {
	register("/go9p/srv/"+conn.Srv.Id+"/conn/"+conn.Id, conn)
}

func (conn *Conn) statsUnregister() {
	register("/go9p/srv/"+conn.Srv.Id+"/conn/"+conn.Id, nil)
}

func (conn *Conn) ServeHTTP(c http.ResponseWriter, r *http.Request) {
	io.WriteString(c, fmt.Sprintf("<html><body><h1>Connection %s/%s</h1>", conn.Srv.Id, conn.Id))
	defer io.WriteString(c, "</body></html>")

	// statistics
	conn.Lock()
	io.WriteString(c, fmt.Sprintf("<p>Number of processed requests: %d", conn.nreqs))
	io.WriteString(c, fmt.Sprintf("<br>Sent %v bytes", conn.rsz))
	io.WriteString(c, fmt.Sprintf("<br>Received %v bytes", conn.tsz))
	io.WriteString(c, fmt.Sprintf("<br>Pending requests: %d max %d", conn.npend, conn.maxpend))
	io.WriteString(c, fmt.Sprintf("<br>Number of reads: %d", conn.nreads))
	io.WriteString(c, fmt.Sprintf("<br>Number of writes: %d", conn.nwrites))
	conn.Unlock()

	// fcalls
	if conn.Debuglevel&DbgLogFcalls != 0 {

	}
}

func StatsHandler(c http.ResponseWriter, r *http.Request) {
	mux.RLock()
	if v, ok := stat[r.URL.Path]; ok {
		v.ServeHTTP(c, r)
	} else if r.URL.Path == "/go9p/" {
		io.WriteString(c, fmt.Sprintf("<html><body><br><h1>On offer: </h1><br>"))
		for v := range stat {
			io.WriteString(c, fmt.Sprintf("<a href='%s'>%s</a><br>", v, v))
		}
		io.WriteString(c, "</body></html>")
	}
	mux.RUnlock()
}
