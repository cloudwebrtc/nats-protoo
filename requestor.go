package nprotoo

import (
	"encoding/json"

	"github.com/cloudwebrtc/nats-protoo/logger"
	nats "github.com/nats-io/nats.go"
)

// Requestor .
type Requestor struct {
	subj         string
	reply        string
	nc           *nats.Conn
	np           *NatsProtoo
	transcations map[int]*Transcation
}

func newRequestor(channel string, np *NatsProtoo, nc *nats.Conn) *Requestor {
	var req Requestor
	req.subj = channel
	req.np = np
	req.nc = nc
	// Sub reply inbox.
	random, _ := GenerateRandomString(12)
	req.reply = "requestor-id-" + random
	req.nc.QueueSubscribe(req.reply, _EMPTY_, req.onReply)
	req.nc.Flush()
	req.transcations = make(map[int]*Transcation)
	return &req
}

// Request .
func (req *Requestor) Request(method string, data map[string]interface{}, success AcceptFunc, reject RejectFunc) {
	id := GenerateRandomNumber()
	request := &Request{
		Request: true,
		ID:      id,
		Method:  method,
		Data:    data,
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

	req.transcations[id] = transcation
	logger.Debugf("Send request [%s]", method)
	req.np.Send(payload, req.subj, req.reply)
}

// SyncRequest .
func (req *Requestor) SyncRequest(method string, data map[string]interface{}) (map[string]interface{}, *Error) {
	return req.AsyncRequest(method, data).Await()
}

// AsyncRequest .
func (req *Requestor) AsyncRequest(method string, data map[string]interface{}) *Future {
	var future = NewFuture()
	req.Request(method, data,
		func(data map[string]interface{}) {
			logger.Debugf("RequestAsFuture: accept [%v]", data)
			future.resolve(data)
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
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		panic(err)
	}
	if data["response"] != nil {
		req.handleResponse(data)
	}
	return
}

func (req *Requestor) handleResponse(response map[string]interface{}) {
	id := int(response["id"].(float64))

	transcation := req.transcations[id]

	if transcation == nil {
		logger.Errorf("received response does not match any sent request [id:%d]", id)
		return
	}

	if response["ok"] != nil && response["ok"] == true {
		transcation.accept(response["data"].(map[string]interface{}))
	} else {
		transcation.reject(int(response["errorCode"].(float64)), response["errorReason"].(string))
	}

	delete(req.transcations, id)
}
