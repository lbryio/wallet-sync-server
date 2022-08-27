package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/wallet"

	"github.com/gorilla/websocket"
)

// Using this as a guide:
// https://github.com/gorilla/websocket/blob/master/examples/chat/
//
// Skipping some things that seem like maybe overkill for a simple application,
// given that this isn't mission critical, and given that I'm not sure what a
// lot of it does. In particular the wsWriter ping stuff. But, we can add it if
// the performance is bad.

const pongWait = 60 * time.Second
const writeWait = 10 * time.Second

type wsClientNotifyType int

const (
	// The channel is closed (i.e. this is the zero-value), so the socket should
	// be closed too. We might not actually check for a closed channel using
	// this value, but it's here for completeness.
	wsClientNotifyFinish = wsClientNotifyType(iota)

	// Inform the client about a wallet update
	wsClientNotifyUpdate
)

// wsClientNotifyMsg is sent over wsClient.notify by the websocket manager
type wsClientNotifyMsg struct {
	notifyType wsClientNotifyType
	sequence   wallet.Sequence
}

const notifyChanBuffer = 5 // Each client shouldn't be getting a lot of concurrent messages

// Given a wsClientNotifyMsg of type wsClientNotifyUpdate, turn it into an
// appropriate message to the client to be sent over websocket
func walletUpdateWSMessage(msg wsClientNotifyMsg) []byte {
	return []byte(fmt.Sprintf("wallet-update:%d", msg.sequence))
}

// Poor man's debug log
const debugWebsockets = false

func debugLog(format string, v ...any) {
	if debugWebsockets {
		log.Printf(format, v...)
	}
}

// Represents a connection to a client.
type wsClient struct {
	socket *websocket.Conn
	notify chan wsClientNotifyMsg
}

// Each user with at least one actively connected client will have one of these
// associated.
type wsClientSet map[*wsClient]bool

// A message sent over a channel to indicate that the given client is
// connecting or disconnecting for the given user.
type wsClientForUser struct {
	userId auth.UserId
	client *wsClient
}

var upgrader = websocket.Upgrader{} // use default options

// Just handle ping/pong
func (s *Server) wsReader(userId auth.UserId, client *wsClient) {
	defer func() {
		// Since wsWriter is waiting on the notify channel, tell the manager to
		// close it. This will make wsWriter stop (if it hasn't already).
		s.clientRemove <- wsClientForUser{userId, client}
		client.socket.Close()

		debugLog("Done with wsReader %+v", client)
	}()

	client.socket.SetReadLimit(512)
	client.socket.SetReadDeadline(time.Now().Add(pongWait))
	client.socket.SetPongHandler(func(string) error { client.socket.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := client.socket.ReadMessage()
		if err != nil {
			debugLog("wsReader: %s\n", err.Error())
			break
		}
	}
}

func (s *Server) wsWriter(userId auth.UserId, client *wsClient) {
	defer func() {
		// Whatever the cause of closure here, closing the socket (if it's not
		// closed already) will cause wsReader to stop (if it hasn't stopped
		// already) since it's waiting on the socket.
		client.socket.Close()

		debugLog("Done with wsWriter %+v", client)
	}()

	for notifyMsg := range client.notify {
		if notifyMsg.notifyType != wsClientNotifyUpdate {
			log.Printf("wsWriter: Got an unknown message type! %+v", notifyMsg)
			continue
		}
		debugLog("wsWriter: notify update")
		client.socket.SetWriteDeadline(time.Now().Add(writeWait))
		err := client.socket.WriteMessage(websocket.TextMessage, walletUpdateWSMessage(notifyMsg))
		if err != nil {
			debugLog("wsWriter: %s\n", err.Error())
			return // skip close message
		}
	}

	// Not sure what the point of this is, given that this probably
	// wouldn't get triggered unless the socket already closed, but the
	// example did this.
	debugLog("wsWriter: sending CloseMessage")
	client.socket.SetWriteDeadline(time.Now().Add(writeWait))
	client.socket.WriteMessage(websocket.CloseMessage, []byte{})
}

// This is the server endpoint that initiates a new websocket
func (s *Server) websocket(w http.ResponseWriter, req *http.Request) {
	token, paramsErr := getTokenParam(req)

	if paramsErr != nil {
		// In this specific case, the error is limited to values that are safe to
		// give to the user.
		errorJson(w, http.StatusBadRequest, paramsErr.Error())
		return
	}

	authToken := s.checkAuth(w, token, auth.ScopeFull)

	if authToken == nil {
		return
	}

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := wsClient{ws, make(chan wsClientNotifyMsg, notifyChanBuffer)}
	newClient := wsClientForUser{authToken.UserId, &client}
	s.clientAdd <- newClient

	go s.wsReader(authToken.UserId, &client)
	go s.wsWriter(authToken.UserId, &client)

	log.Println("Client Connected")
}

func (s *Server) manageSockets(done chan bool, finish chan bool) {
	log.Println("Socket manager start")
	clientsByUser := make(map[auth.UserId]wsClientSet)

	removeClient := func(userId auth.UserId, client *wsClient) {
		debugLog("removeClient %+v", client)
		if _, ok := clientsByUser[userId]; !ok {
			return
		}
		if _, ok := clientsByUser[userId][client]; !ok {
			return
		}

		close(client.notify)
		delete(clientsByUser[userId], client)

		if len(clientsByUser[userId]) == 0 {
			delete(clientsByUser, userId)
		}
	}

	removeUser := func(userId auth.UserId) {
		debugLog("removeUser (which calls removeClient) %d", userId)

		for client := range clientsByUser[userId] {
			removeClient(userId, client)
		}
	}

	addClient := func(userId auth.UserId, client *wsClient) {
		debugLog("addClient %+v", client)
		if _, ok := clientsByUser[userId]; !ok {
			clientsByUser[userId] = make(wsClientSet)
		}
		clientsByUser[userId][client] = true
	}

manage:
	for {
		select {
		case msg := <-s.walletUpdates:
			for client := range clientsByUser[msg.userId] {
				select {
				case client.notify <- wsClientNotifyMsg{wsClientNotifyUpdate, msg.sequence}:
				default:
					log.Println("This is a bug: Channel was somehow closed but the manager has not (yet) received a clientRemove message.")

					// The example program had this, but I don't see why.
					removeClient(msg.userId, client)
				}
			}
		case removedUser := <-s.userRemove:
			removeUser(removedUser.userId)
		case retiredClient := <-s.clientRemove:
			removeClient(retiredClient.userId, retiredClient.client)
		case newClient := <-s.clientAdd:
			addClient(newClient.userId, newClient.client)
		case <-finish:
			break manage
		}
	}

	log.Println("Cleaning up sockets")

	debugLog("Running any addClient messages that snuck in...")

	// By the time the `finish` channel has triggered, the web server has shut
	// down, so we won't have any clientAdd events _triggered_ by this point.
	// However, one (or more, if we add a buffer later) may be in the queue, so
	// let's keep track of them here so we can close them. But we close the
	// clientAdd channel first so we can break out of this loop.

	// We assume that the server is done writing at this point, thus it's safe
	// to close this channel here.
	close(s.clientAdd)
	for newClient := range s.clientAdd {
		addClient(newClient.userId, newClient.client)
	}

	// Now that we know about every running client, just close all of the sockets
	// (double-closing seems to be safe, so we don't care about races here). But
	// if it takes more than 10 seconds for whatever reason, just bail.

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		select {
		case <-ticker.C:
			log.Println("Giving up on closing remaining sockets cleanly.")

			// This will signal to main to exit, which will end the program
			done <- true

			log.Println("Socket manager impolite finish")
		}
	}()

	debugLog("Closing sockets...")
	for _, userClients := range clientsByUser {
		for client := range userClients {
			debugLog("Closing socket for %+v", client)
			client.socket.SetWriteDeadline(time.Now().Add(writeWait))
			client.socket.WriteMessage(websocket.CloseMessage, []byte{})
			// TODO - wait for receiving the CloseMessage?
			client.socket.Close()
			debugLog("Closed socket for %+v", client)
		}
	}

	// TODO - Do we need to wait for the sockets to actually close after
	// calling Close(), before exiting the program? Alternately: do they close
	// automatically anyway and I don't need this cleanup stuff in the first
	// place? (Probably doesn't automatically send the Close message at least.)

	done <- true
	log.Println("Socket manager finish")
}
