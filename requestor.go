package nprotoo

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/chuckpreslar/emission"
	"github.com/cloudwebrtc/nats-protoo/logger"
	nats "github.com/nats-io/nats.go"
)

const (
	// DefaultRequestTimeout .
	DefaultRequestTimeout = 15 * time.Second
)

// Requestor .
type Requestor struct {
	emission.Emitter
	subj         string
	reply        string
	nc           *nats.Conn
	np           *NatsProtoo
	timeout      time.Duration
	transcations map[int]*Transcation
	mutex        *sync.Mutex
}

func newRequestor(channel string, np *NatsProtoo, nc *nats.Conn) *Requestor {
	var req Requestor
	req.Emitter = *emission.NewEmitter()
	req.mutex = new(sync.Mutex)
	req.subj = channel
	req.np = np
	req.timeout = DefaultRequestTimeout
	req.np.On("close", func(code int, err string) {
		logger.Infof("Transport closed [%d] %s", code, err)
		req.Emit("close", code, err)
	})
	req.np.On("error", func(code int, err string) {
		logger.Warnf("Transport got error (%d, %s)", code, err)
		req.Emit("error", code, err)
	})
	req.nc = nc
	// Sub reply inbox.
	random, _ := GenerateRandomString(12)
	req.reply = "requestor-id-" + random
	req.nc.QueueSubscribe(req.reply, _EMPTY_, req.onReply)
	req.nc.Flush()
	req.transcations = make(map[int]*Transcation)
	return &req
}

// SetRequestTimeout .
func (req *Requestor) SetRequestTimeout(d time.Duration) {
	req.mutex.Lock()
	defer req.mutex.Unlock()
	req.timeout = d
}

// Request .
func (req *Requestor) Request(method string, data interface{}, success AcceptFunc, reject RejectFunc) {
	id := GenerateRandomNumber()
	dataStr, err := json.Marshal(data)
	if err != nil {
		logger.Errorf("Marshal data %v", err)
		return
	}
	request := &Request{
		RequestData: RequestData{
			Request: true,
		},
		CommonData: CommonData{
			ID:     id,
			Method: method,
			Data:   dataStr,
		},
	}
	payload, err := json.Marshal(request)
	if err != nil {
		logger.Errorf("Marshal %v", err)
		return
	}

	transcation := &Transcation{
		id:     id,
		accept: success,
		reject: reject,
		close: func() {
			logger.Infof("Transport closed !")
		},
	}

	{
		req.mutex.Lock()
		defer req.mutex.Unlock()
		req.transcations[id] = transcation
		transcation.timer = time.AfterFunc(req.timeout, func() {
			logger.Debugf("Request timeout transcation[%d]", transcation.id)
			transcation.reject(480, fmt.Sprintf("Request timeout %fs transcation[%d], method[%s]", req.timeout.Seconds(), transcation.id, method))
			req.mutex.Lock()
			defer req.mutex.Unlock()
			delete(req.transcations, id)
		})
	}

	logger.Debugf("Send request [%s]", method)
	req.np.Send(payload, req.subj, req.reply)
}

// SyncRequest .
func (req *Requestor) SyncRequest(method string, data interface{}) (RawMessage, *Error) {
	return req.AsyncRequest(method, data).Await()
}

// AsyncRequest .
func (req *Requestor) AsyncRequest(method string, data interface{}) *Future {
	var future = NewFuture()
	req.Request(method, data,
		func(resultData RawMessage) {
			logger.Debugf("RequestAsFuture: accept [%v]", data)
			future.resolve(resultData)
		},
		func(code int, reason string) {
			logger.Debugf("RequestAsFuture: reject [%d:%s]", code, reason)
			future.reject(&Error{code, reason})
		})
	return future
}

func (req *Requestor) onReply(msg *nats.Msg) {
	logger.Debugf("Got response [subj:%s, reply:%s]: %s", msg.Subject, msg.Reply, string(msg.Data))
	req.handleMessage(msg.Data, msg.Subject, msg.Reply)
}

func (req *Requestor) handleMessage(message []byte, subj string, reply string) {
	var msg PeerMsg
	if err := json.Unmarshal(message, &msg); err != nil {
		logger.Errorf("handleMessage PeerMsg Unmarshal %v", err)
		return
	}

	if msg.Response {
		var data Response
		if err := json.Unmarshal(message, &data); err != nil {
			logger.Errorf("handleMessage Response Unmarshal %v", err)
			return
		}
		req.handleResponse(data)
	}
}

func (req *Requestor) handleResponse(response Response) {
	req.mutex.Lock()
	defer req.mutex.Unlock()
	transcation := req.transcations[response.ID]

	if transcation == nil {
		logger.Errorf("received response does not match any sent request [id:%d]", response.ID)
		return
	}

	transcation.timer.Stop()

	if response.Ok {
		transcation.accept(response.Data)
	} else {
		transcation.reject(response.ErrorCode, response.ErrorReason)
	}

	delete(req.transcations, response.ID)
}
