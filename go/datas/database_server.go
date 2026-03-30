// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/d"
	"github.com/julienschmidt/httprouter"
)

type connectionState struct {
	c  net.Conn
	cs http.ConnState
}

var allowedOrigins []string

func SetAllowedOrigins(origins []string) {
	allowedOrigins = origins
}

func isOriginAllowed(origin string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(origin, allowed) {
			return true
		}
		if strings.HasSuffix(allowed, ".*") {
			domain := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}

type RemoteDatabaseServer struct {
	cs      chunks.ChunkStore
	address string
	port    int
	l       *net.Listener
	csChan  chan *connectionState
	closing bool
	Ready   func()
}

func NewRemoteDatabaseServer(cs chunks.ChunkStore, address string, port int) *RemoteDatabaseServer {
	dataVersion := cs.Version()
	if constants.NomsVersion != dataVersion {
		d.Panic("SDK version %s is incompatible with data of version %s", constants.NomsVersion, dataVersion)
	}
	return &RemoteDatabaseServer{
		cs:      cs,
		address: address,
		port:    port,
		l:       nil,
		csChan:  make(chan *connectionState, 16),
		closing: false,
		Ready:   func() {},
	}
}

// Port is the actual port used. This may be different than the port passed in to NewRemoteDatabaseServer.
func (s *RemoteDatabaseServer) Port() int {
	return s.port
}

func Router(cs chunks.ChunkStore, prefix string) *httprouter.Router {
	router := httprouter.New()

	router.POST(prefix+constants.GetRefsPath, corsHandle(makeHandle(HandleGetRefs, cs)))
	router.GET(prefix+constants.GetBlobPath, corsHandle(makeHandle(HandleGetBlob, cs)))
	router.OPTIONS(prefix+constants.GetRefsPath, corsHandle(noopHandle))
	router.POST(prefix+constants.HasRefsPath, corsHandle(makeHandle(HandleHasRefs, cs)))
	router.OPTIONS(prefix+constants.HasRefsPath, corsHandle(noopHandle))
	router.GET(prefix+constants.RootPath, corsHandle(makeHandle(HandleRootGet, cs)))
	router.POST(prefix+constants.RootPath, corsHandle(makeHandle(HandleRootPost, cs)))
	router.OPTIONS(prefix+constants.RootPath, corsHandle(noopHandle))
	router.POST(prefix+constants.WriteValuePath, corsHandle(makeHandle(HandleWriteValue, cs)))
	router.OPTIONS(prefix+constants.WriteValuePath, corsHandle(noopHandle))
	router.GET(prefix+constants.BasePath, corsHandle(makeHandle(HandleBaseGet, cs)))

	router.GET(prefix+constants.GraphQLPath, corsHandle(makeHandle(HandleGraphQL, cs)))
	router.POST(prefix+constants.GraphQLPath, corsHandle(makeHandle(HandleGraphQL, cs)))
	router.OPTIONS(prefix+constants.GraphQLPath, corsHandle(noopHandle))

	router.GET(prefix+constants.StatsPath, corsHandle(makeHandle(HandleStats, cs)))
	router.OPTIONS(prefix+constants.StatsPath, corsHandle(noopHandle))

	return router
}

// Run blocks while the RemoteDatabaseServer is listening. Running on a separate go routine is supported.
func (s *RemoteDatabaseServer) Run() {

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.address, s.port))
	d.Chk.NoError(err)
	s.l = &l
	_, port, err := net.SplitHostPort(l.Addr().String())
	d.Chk.NoError(err)
	s.port, err = strconv.Atoi(port)
	d.Chk.NoError(err)
	log.Printf("Listening on  %s:%d...\n", s.address, s.port)

	router := Router(s.cs, "")

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			router.ServeHTTP(w, req)
		}),
		ConnState: s.connState,
	}

	go func() {
		m := map[net.Conn]http.ConnState{}
		for connState := range s.csChan {
			switch connState.cs {
			case http.StateNew, http.StateActive, http.StateIdle:
				m[connState.c] = connState.cs
			default:
				delete(m, connState.c)
			}
		}
		for c := range m {
			c.Close()
		}
	}()

	go s.Ready()
	srv.Serve(l)
}

func makeHandle(hndlr Handler, cs chunks.ChunkStore) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		hndlr(w, req, ps, cs)
	}
}

func noopHandle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
}

func corsHandle(f httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		origin := r.Header.Get("Origin")
		if origin != "" && isOriginAllowed(origin) {
			w.Header().Add("Access-Control-Allow-Origin", origin)
			w.Header().Add("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Noms-Version")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Add("Access-Control-Expose-Headers", NomsVersionHeader)
		w.Header().Add(NomsVersionHeader, constants.NomsVersion)
		f(w, r, ps)
	}
}

func (s *RemoteDatabaseServer) connState(c net.Conn, cs http.ConnState) {
	if s.closing {
		d.PanicIfFalse(cs == http.StateClosed)
		return
	}
	s.csChan <- &connectionState{c, cs}
}

// Will cause the RemoteDatabaseServer to stop listening and an existing call to Run() to continue.
func (s *RemoteDatabaseServer) Stop() {
	s.closing = true
	(*s.l).Close()
	(s.cs).Close()
	close(s.csChan)
}
