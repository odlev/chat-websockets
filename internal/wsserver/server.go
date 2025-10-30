// Package wsserver is a nice package
package wsserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/odlev/websockets/internal/config"
	"github.com/rs/zerolog"
)

type WSServer interface {
	Start() error
	Address() string
	Stop() error
}

type wsSrv struct {
	mux       *http.ServeMux
	srv       *http.Server
	Upgrader  websocket.Upgrader
	cfg       *config.Config
	Clients   map[*websocket.Conn]struct{}
	mu        sync.RWMutex
	broadcast chan *wsMessage
	log       zerolog.Logger
	certFile  string
	keyFile   string
}

func New(addr string, cfg *config.Config, log zerolog.Logger) WSServer {
	m := http.NewServeMux()
	return &wsSrv{
		mux: m,
		srv: &http.Server{
			Addr:    addr,
			Handler: m,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
		},
		Upgrader:  websocket.Upgrader{},
		cfg:       cfg,
		Clients:   make(map[*websocket.Conn]struct{}),
		broadcast: make(chan *wsMessage),
		log:       log,
		certFile:  cfg.Certificates.CertificatePath,
		keyFile:   cfg.Certificates.KeyPath,
	}
}

func (ws *wsSrv) Start() error {
	cert, err := tls.LoadX509KeyPair(ws.certFile, ws.keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate pair: %w", err)
	}

	ws.srv.TLSConfig.Certificates = append(ws.srv.TLSConfig.Certificates, cert)
	ws.mux.Handle("/", http.FileServer(http.Dir(ws.cfg.HTMLAddress)))

	ws.mux.HandleFunc("/test", ws.testHandler)
	ws.mux.HandleFunc("/ws", ws.wsHandler)
	go ws.writeToclientsBroadcast()

	return ws.srv.ListenAndServeTLS(ws.certFile, ws.keyFile)
}

func (ws *wsSrv) Address() string {
	return ws.srv.Addr
}
func (ws *wsSrv) testHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("test is successful"))
}

func (ws *wsSrv) Stop() error {
	close(ws.broadcast)
	ws.mu.Lock()
	for conn := range ws.Clients {
		c := conn // дабы избежать closure (замыкания) и не создавать go func() {}()
		go c.Close()
	}
	clear(ws.Clients)
	ws.mu.Unlock()
	if err := ws.srv.Shutdown(context.Background()); err != nil {
		return err
	}
	return nil
}

func (ws *wsSrv) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.Upgrader.Upgrade(w, r, http.Header{"hello": {"hello too"}})
	if err != nil {
		ws.log.Err(err).Msg("error with websocket connection")
		return
	}
	ws.mu.Lock()
	ws.log.Info().Str("client ip address", conn.RemoteAddr().String()).Send()
	ws.mu.Unlock()
	ws.Clients[conn] = struct{}{}

	go ws.readFromClient(conn)
}

// readFromClient reads message and sends *wsMessage to broadcast channel
func (ws *wsSrv) readFromClient(conn *websocket.Conn) {
	for {
		msg := &wsMessage{}
		if err := conn.ReadJSON(msg); err != nil {
			ws.log.Err(err).Msg("failed to read json")
			break
		}
		host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			ws.log.Err(err).Msg("failed to split client addr into host and port")
		}
		msg.IPAddress = host
		msg.Time = time.Now().Format("15:04")

		ws.broadcast <- msg
	}
	ws.mu.Lock()
	delete(ws.Clients, conn)
	ws.mu.Unlock()
}

func (ws *wsSrv) writeToclientsBroadcast() {
	for msg := range ws.broadcast {
		ws.mu.RLock() // читает из мапы (!не пишет в мапу, поэтому рлок) и пишет в вебсокет соединение каждому клиенту
		for client := range ws.Clients {
			go func(c *websocket.Conn) {
				if err := c.WriteJSON(msg); err != nil {
					ws.log.Err(err).Msg("failed to write message")
				}
			}(client)
		}
		ws.mu.RUnlock()
	}
}
