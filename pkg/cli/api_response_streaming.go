package cli

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"net/url"
)

var _ APIResponse = &StreamingApiResponse{}

type StreamingApiResponseHandler func(ctx context.Context, conn *websocket.Conn, msg chan []byte, err chan error)

type StreamingApiResponse struct {
	url           *url.URL
	streamContext context.Context
	cancel        context.CancelFunc
	messageChan   chan []byte
	errorChan     chan error
	streamingFunc StreamingApiResponseHandler
	dialer        *websocket.Dialer
}

func NewStreamingApiResponse(url *url.URL, dialer *websocket.Dialer, streamingFunc StreamingApiResponseHandler) APIResponse {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamingApiResponse{
		url:           url,
		streamContext: ctx,
		cancel:        cancel,
		messageChan:   make(chan []byte),
		errorChan:     make(chan error),
		streamingFunc: streamingFunc,
		dialer:        dialer,
	}
}

func (resp *StreamingApiResponse) Err() error {
	return nil
}

func (resp *StreamingApiResponse) Print() error {
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
