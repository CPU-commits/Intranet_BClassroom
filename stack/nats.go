package stack

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/settings"
	"github.com/nats-io/nats.go"
)

const QUEUE_NAME = "classroom"

type NatsClient struct {
	conn *nats.Conn
}

// Nats NESTJS
type NatsNestJSRes struct {
	ID         string      `json:"id"`
	IsDisposed bool        `json:"isDisposed"`
	Response   interface{} `json:"response"`
}

// Nats Golang
type NatsGolangReq struct {
	Pattern string      `json:"pattern"`
	Data    interface{} `json:"data"`
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

func (nats *NatsClient) DecodeDataNest(data []byte) (map[string]interface{}, error) {
	var dataNest NatsGolangReq

	err := json.Unmarshal(data, &dataNest)
	if err != nil {
		return nil, err
	}
	payload := make(map[string]interface{})
	v := reflect.ValueOf(dataNest.Data)
	if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			strct := v.MapIndex(key)
			payload[key.String()] = strct.Interface()
		}
	} else {
		return nil, fmt.Errorf("data not is a map")
	}
	return payload, nil
}

func (nats *NatsClient) Subscribe(channel string, toDo func(m *nats.Msg)) {
	nats.conn.Subscribe(channel, toDo)
}

func (nats *NatsClient) Publish(channel string, message []byte) {
	nats.conn.Publish(channel, message)
}

func (client *NatsClient) Queue(channel string, toDo func(m *nats.Msg)) {
	client.conn.QueueSubscribe(channel, QUEUE_NAME, toDo)
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
