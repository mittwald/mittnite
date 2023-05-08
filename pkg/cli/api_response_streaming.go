package cli

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"net/url"
)

var _ APIResponse = &StreamingAPIResponse{}

type StreamingAPIResponseHandler func(ctx context.Context, conn *websocket.Conn, msg chan []byte, err chan error)

type StreamingAPIResponse struct {
	url           *url.URL
	streamContext context.Context
	cancel        context.CancelFunc
	messageChan   chan []byte
	errorChan     chan error
	streamingFunc StreamingAPIResponseHandler
	dialer        *websocket.Dialer
}

func NewStreamingAPIResponse(url *url.URL, dialer *websocket.Dialer, streamingFunc StreamingAPIResponseHandler) APIResponse {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamingAPIResponse{
		url:           url,
		streamContext: ctx,
		cancel:        cancel,
		messageChan:   make(chan []byte),
		errorChan:     make(chan error),
		streamingFunc: streamingFunc,
		dialer:        dialer,
	}
}

func (resp *StreamingAPIResponse) Err() error {
	return nil
}

func (resp *StreamingAPIResponse) Print() error {
	conn, _, err := resp.dialer.Dial(resp.url.String(), nil)
	if err != nil {
		return fmt.Errorf("error dialing to %s: %w", resp.url.String(), err)
	}
	defer func() {
		resp.cancel()
		close(resp.messageChan)
		close(resp.errorChan)
		conn.Close()
	}()

	go resp.streamingFunc(resp.streamContext, conn, resp.messageChan, resp.errorChan)

	for {
		select {
		case msg := <-resp.messageChan:
			fmt.Println(string(msg))
		case err := <-resp.errorChan:
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				return nil
			}
			return err
		case <-resp.streamContext.Done():
			return nil
		}
	}
}
