package nprotoo

import (
	"encoding/json"
	"errors"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/chuckpreslar/emission"
	"github.com/cloudwebrtc/nats-protoo/logger"
	nats "github.com/nats-io/nats.go"
)

const (
	pingPeriod     = 5 * time.Second
	DefaultNatsURL = "nats://127.0.0.1:4222"
	_EMPTY_        = ""
)

// NatsProtoo .
type NatsProtoo struct {
	emission.Emitter
	nc        *nats.Conn
	mutex     *sync.Mutex
	subj      string
	closed    bool
	listeners map[string][]interface{}
}

// NewNatsProtoo .
func NewNatsProtoo(server string) *NatsProtoo {
	var np NatsProtoo
	np.closed = false
	// Connect Options.
	opts := []nats.Option{nats.Name("NATS Protoo")}
	opts = setupConnOptions(opts)
	// Connect to NATS
	nc, err := nats.Connect(server, opts...)
	if err != nil {
		log.Fatal(err)
	}
	np.nc = nc
	np.nc.SetClosedHandler(func(nc *nats.Conn) {
		logger.Warnf("%s [%v]", "nats nc closed", nc.LastError)
		np.Emit("close", 0, nc.LastError().Error)
		np.closed = true
	})
	np.Emitter = *emission.NewEmitter()
	np.mutex = new(sync.Mutex)
	np.listeners = make(map[string][]interface{})
	logger.Infof("New Nats Protoo: nats => %s", server)
	return &np
}

func (np *NatsProtoo) NewRequestor(channel string) *Requestor {
	return newRequestor(channel, np, np.nc)
}

func (np *NatsProtoo) OnRequest(channel string, listener RequestFunc) {
	np.mutex.Lock()
	defer np.mutex.Unlock()
	if _, found := np.listeners[channel]; !found {
		np.nc.QueueSubscribe(channel, _EMPTY_, np.onRequest)
		np.nc.Flush()
	}
	listeners, _ := np.listeners[channel]
	if !isContain(listeners, listener) {
		np.On("request", listener)
		logger.Debugf("OnRequest: [channel:%s, listener:%v]", channel, listener)
		np.listeners[channel] = append(listeners, listener)
	}
}

func (np *NatsProtoo) NewBroadcaster(channel string) *Broadcaster {
	return newBroadcaster(channel, np, np.nc)
}

func (np *NatsProtoo) OnBroadcast(channel string, listener func(data map[string]interface{}, subj string)) {
	np.mutex.Lock()
	defer np.mutex.Unlock()
	if _, found := np.listeners[channel]; !found {
		np.nc.QueueSubscribe(channel, _EMPTY_, np.onRequest)
		np.nc.Flush()
	}
	listeners, _ := np.listeners[channel]
	if !isContain(listeners, listener) {
		np.On("broadcast", listener)
		logger.Debugf("OnBroadcast: [channel:%s, listener:%v]", channel, listener)
		np.listeners[channel] = append(listeners, listener)
	}
}

func (np *NatsProtoo) onRequest(msg *nats.Msg) {
	logger.Debugf("Got request [subj:%s, reply:%s]: %s", msg.Subject, msg.Reply, string(msg.Data))
	np.handleMessage(msg.Data, msg.Subject, msg.Reply)
}

func (np *NatsProtoo) handleMessage(message []byte, subj string, reply string) {
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		logger.Errorf("np.handleMessage error => %v", err)
	}
	if data["request"] != nil {
		np.handleRequest(data, subj, reply)
	} else if data["notification"] != nil {
		np.handleBroadcast(data, subj, reply)
	}
	return
}

func (np *NatsProtoo) handleRequest(request map[string]interface{}, subj string, reply string) {
	logger.Debugf("Handle request [%s]", request["method"])
	accept := func(data map[string]interface{}) {
		response := &Response{
			Response: true,
			Ok:       true,
			ID:       int(request["id"].(float64)),
			Data:     data,
		}
		payload, err := json.Marshal(response)
		if err != nil {
			logger.Errorf("Marshal %v", err)
			return
		}
		//send accept
		logger.Debugf("Accept [%s] => (%s)", request["method"], payload)
		np.Reply(payload, reply)
	}

	reject := func(errorCode int, errorReason string) {
		response := &ResponseError{
			Response:    true,
			Ok:          false,
			ID:          int(request["id"].(float64)),
			ErrorCode:   errorCode,
			ErrorReason: errorReason,
		}
		payload, err := json.Marshal(response)
		if err != nil {
			logger.Errorf("Marshal %v", err)
			return
		}
		//send reject
		logger.Debugf("Reject [%s] => (errorCode:%d, errorReason:%s)", request["method"], errorCode, errorReason)
		np.Reply(payload, reply)
	}

	np.Emit("request", request, accept, reject)
}

func (np *NatsProtoo) handleBroadcast(data map[string]interface{}, subj string, reply string) {
	np.Emit("broadcast", data, subj)
}

// Close .
func (np *NatsProtoo) Close() {
	np.mutex.Lock()
	defer np.mutex.Unlock()
	if np.closed == false {
		logger.Infof("Close nats nc now : ", np)
		np.nc.Close()
		np.closed = true
	} else {
		logger.Warnf("Transport already closed :", np)
	}
}

// Send .
func (np *NatsProtoo) Send(message []byte, subj string, reply string) error {
	logger.Debugf("Send: %s", string(message))
	np.mutex.Lock()
	defer np.mutex.Unlock()
	if np.closed {
		return errors.New("websocket: write closed")
	}
	err := np.nc.PublishRequest(subj, reply, message)
	if err != nil {
		if np.nc.LastError() != nil {
			log.Fatalf("%v for request", np.nc.LastError())
		}
		log.Fatalf("%v for request", err)
		return err
	}
	return nil
}

// Reply .
func (np *NatsProtoo) Reply(message []byte, reply string) error {
	logger.Debugf("Reply: %s", string(message))
	np.mutex.Lock()
	defer np.mutex.Unlock()
	if np.closed {
		return errors.New("websocket: write closed")
	}
	err := np.nc.Publish(reply, message)
	if err != nil {
		if np.nc.LastError() != nil {
			log.Fatalf("%v for request", np.nc.LastError())
		}
		log.Fatalf("%v for request", err)
		return err
	}
	return nil
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		log.Printf("Disconnected due to: %s, will attempt reconnects for %.0fm", err, totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Fatalf("Exiting: %v", nc.LastError())
	}))

	return opts
}

func isContain(items []interface{}, item interface{}) bool {
	for _, eachItem := range items {
		if reflect.ValueOf(eachItem) == reflect.ValueOf(item) {
			return true
		}
	}
	return false
}
