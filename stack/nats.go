package stack

import (
	"fmt"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/settings"
	"github.com/nats-io/nats.go"
)

type NatsClient struct {
	conn *nats.Conn
}

// Nats NESTJS
type NatsNestJSRes struct {
	ID         string      `json:"id"`
	IsDisposed bool        `json:"isDisposed"`
	Response   interface{} `json:"response"`
}

var settingsData = settings.GetSettings()

func newConnection() *nats.Conn {
	uriNats := fmt.Sprintf("nats://%s:4222", settingsData.NATS_HOST)
	nc, err := nats.Connect(uriNats)
	if err != nil {
		panic(err)
	}
	return nc
}

func (nats *NatsClient) Subscribe(channel string, toDo func(m *nats.Msg)) {
	nats.conn.Subscribe(channel, toDo)
}

func (nats *NatsClient) Publish(channel string, message []byte) {
	nats.conn.Publish(channel, message)
}

func (nats *NatsClient) Request(channel string, data []byte) (*nats.Msg, error) {
	msg, err := nats.conn.Request(channel, data, time.Second*10)
	return msg, err
}

func (client *NatsClient) PublishEncode(channel string, jsonData interface{}) error {
	ec, err := nats.NewEncodedConn(client.conn, nats.JSON_ENCODER)
	if err != nil {
		return err
	}
	if err := ec.Publish(channel, jsonData); err != nil {
		return err
	}
	return nil
}

func (client *NatsClient) RequestEncode(channel string, jsonData interface{}) (interface{}, error) {
	ec, err := nats.NewEncodedConn(client.conn, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}
	var msg interface{}
	if err := ec.Request(channel, jsonData, msg, time.Second*5); err != nil {
		return nil, err
	}
	return msg, nil
}

func NewNats() *NatsClient {
	conn := newConnection()
	natsClient := &NatsClient{
		conn: conn,
	}
	natsClient.Subscribe("help", func(m *nats.Msg) {
		fmt.Printf("Received a message: %s\n", string(m.Data))
	})
	return natsClient
}
