package main

import (
	"crypto/tls"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func proxyStdio(conn *websocket.Conn, stdin io.Reader, stdout io.Writer) error {
	errCh := make(chan error, 2)
	var closeOnce sync.Once
	closeConn := func() {
		closeOnce.Do(func() {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
			_ = conn.Close()
		})
	}

	go func() {
		errCh <- copyReaderToWebSocket(conn, stdin)
		closeConn()
	}()
	go func() {
		errCh <- copyWebSocketToWriter(conn, stdout)
		closeConn()
	}()

	err := <-errCh
	if err == nil || errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return nil
	}
	return err
}

func copyReaderToWebSocket(conn *websocket.Conn, reader io.Reader) error {
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			return err
		}
	}
}

func copyWebSocketToWriter(conn *websocket.Conn, writer io.Writer) error {
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if messageType != websocket.BinaryMessage && messageType != websocket.TextMessage {
			continue
		}
		if _, err := writer.Write(payload); err != nil {
			return err
		}
	}
}

func newWebSocketDialer(insecure bool) *websocket.Dialer {
	dialer := *websocket.DefaultDialer
	if insecure {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &dialer
}
