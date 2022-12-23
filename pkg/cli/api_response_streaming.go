package cli

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
)

var _ ApiResponse = &StreamingApiResponse{}

type StreamingApiResponseHandler func(ctx context.Context, conn *websocket.Conn, msg chan []byte, err chan error)

type StreamingApiResponse struct {
	url           string
	streamContext context.Context
	cancel        context.CancelFunc
	messageChan   chan []byte
	errorChan     chan error
	streamingFunc StreamingApiResponseHandler
	dialer        *websocket.Dialer
}

func NewStreamingApiResponse(url string, dialer *websocket.Dialer, streamingFunc StreamingApiResponseHandler) ApiResponse {
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

func (resp *StreamingApiResponse) Print() error {
	conn, _, err := resp.dialer.Dial(resp.url, nil)
	if err != nil {
		return err
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
			return err
		case <-resp.streamContext.Done():
			return nil
		}
	}
}
