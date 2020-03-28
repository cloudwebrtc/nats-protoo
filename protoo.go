package nprotoo

import (
	"encoding/json"
	"errors"
	"fmt"
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
	nc                 *nats.Conn
	mutex              *sync.Mutex
	subj               string
	closed             bool
	requestListener    map[string]RequestFunc
	broadcastListeners map[string][]BroadCastFunc
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
	np.requestListener = make(map[string]RequestFunc)
	np.broadcastListeners = make(map[string][]BroadCastFunc)
	logger.Infof("New Nats Protoo: nats => %s", server)
	return &np
}

func (np *NatsProtoo) NewRequestor(channel string) *Requestor {
	return newRequestor(channel, np, np.nc)
}

func (np *NatsProtoo) OnRequest(channel string, listener RequestFunc) {
	np.mutex.Lock()
	defer np.mutex.Unlock()
	if _, found := np.requestListener[channel]; !found {
		np.nc.QueueSubscribe(channel, _EMPTY_, np.onRequest)
		np.nc.Flush()
	}
	np.requestListener[channel] = listener
}

func (np *NatsProtoo) NewBroadcaster(channel string) *Broadcaster {
	return newBroadcaster(channel, np, np.nc)
}

func (np *NatsProtoo) OnBroadcast(channel string, listener BroadCastFunc) {
	np.mutex.Lock()
	defer np.mutex.Unlock()

	if _, found := np.broadcastListeners[channel]; !found {
		np.nc.QueueSubscribe(channel, _EMPTY_, np.onRequest)
		np.nc.Flush()
		np.broadcastListeners[channel] = make([]BroadCastFunc, 0)
	}

	if !listenerIsContain(np.broadcastListeners[channel], listener) {
		logger.Debugf("OnBroadcast: [channel:%s, listener:%v]", channel, listener)
		np.broadcastListeners[channel] = append(np.broadcastListeners[channel], listener)
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

	if listener, found := np.requestListener[subj]; found {
		listener(request, accept, reject)
	} else {
		reject(500, fmt.Sprintf("Not found listener for %s!", subj))
	}
}

func (np *NatsProtoo) handleBroadcast(data map[string]interface{}, subj string, reply string) {

	if listeners, found := np.broadcastListeners[subj]; found {
		for _, listener := range listeners {
			listener(data, subj)
		}
	} else {
		logger.Warnf("handleBroadcast: Not found any callbacks!")
	}
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

func listenerIsContain(items []BroadCastFunc, item BroadCastFunc) bool {
	for _, eachItem := range items {
		if reflect.ValueOf(eachItem) == reflect.ValueOf(item) {
			return true
		}
	}
	return false
}
