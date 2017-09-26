// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/monitor"
)

type Config struct {
	Port       int
	StaticRoot file.Path
	monitor.Managers
}

type Server struct {
	listener listener
	Config

	o monitor.DataOwner
}

type listener struct {
	*net.TCPListener
}

func Create(ctx context.Context, config Config) (*Server, error) {
	server := &Server{Config: config}

	server.o = monitor.NewDataOwner()
	go func() {
		if err := monitor.Run(ctx, server.Managers, server.o, nil); err != nil {
			log.E(ctx, "Monitoring failed")
		}
	}()

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return nil, err
	}
	if config.StaticRoot.IsEmpty() {
		log.I(ctx, "Serving internal web content")
		http.Handle("/", http.FileServer(static("www")))
	} else {
		path := config.StaticRoot.System()
		log.I(ctx, "Serving web content from %s", path)
		http.Handle("/", http.FileServer(http.Dir(path)))
	}
	http.HandleFunc("/tracks/", server.handleTracks)
	http.HandleFunc("/packages/", server.handlePackages)
	http.HandleFunc("/artifacts/", server.handleArtifacts)
	http.HandleFunc("/subjects/", server.handleSubjects)
	http.HandleFunc("/satellites/", server.handleSatellites)
	http.HandleFunc("/traces/", server.handleTraces)
	http.HandleFunc("/replays/", server.handleReplays)
	http.HandleFunc("/reports/", server.handleReports)
	http.HandleFunc("/devices/", server.handleDevices)
	http.HandleFunc("/workers/", server.handleWorkers)
	http.HandleFunc("/entities/", server.handleEntities)
	http.HandleFunc("/status/", server.handleStatus)
	server.listener = listener{l.(*net.TCPListener)}
	return server, nil
}

func (s *Server) Serve(ctx context.Context) error {
	return http.Serve(s.listener, nil)
}

func (s *Server) Close() error {
	return s.listener.Close()
}

func (l listener) Accept() (net.Conn, error) {
	c, err := l.AcceptTCP()
	if err == nil {
		c.SetKeepAlive(true)
		c.SetKeepAlivePeriod(3 * time.Minute)
	}
	return c, err
}
