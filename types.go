package nprotoo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/common/log"
)

type RawMessage json.RawMessage

func (r RawMessage) Unmarshall(msgType interface{}) *Error {
	if err := json.Unmarshal(r, &msgType); err != nil {
		log.Errorf("Biz.Entry parse error %v", err.Error())
		return &Error{Code: 400, Reason: fmt.Sprintf("Error parsing request object %v", err.Error())}
	}
	return nil
}

// AcceptFunc .
type AcceptFunc func(data RawMessage)
type RespondFunc func(data interface{})

// RejectFunc .
type RejectFunc func(errorCode int, errorReason string)

// RequestFunc .
type RequestFunc func(request Request, accept RespondFunc, reject RejectFunc)

// BroadCastFunc .
type BroadCastFunc func(data Notification, subj string)

type PeerMsg struct {
	RequestData
	ResponseData
	NotificationData
	CommonData
}

type RequestData struct {
	Request   bool   `json:"request"`
	ReplySubj string `json:"reply"`
}

type ResponseData struct {
	Response bool `json:"response"`
	Ok       bool `json:"ok"`
	ResponseErrData
}

type ResponseErrData struct {
	ErrorCode   int    `json:"errorCode"`
	ErrorReason string `json:"errorReason"`
}

type NotificationData struct {
	Notification bool `json:"notification"`
}

type CommonData struct {
	ID     int        `json:"id"`
	Method string     `json:"method"`
	Data   RawMessage `json:"data"`
}

func (m PeerMsg) ToNotification() Notification {
	return Notification{NotificationData: m.NotificationData, CommonData: m.CommonData}
}

func (m PeerMsg) ToRequest() Request {
	return Request{RequestData: m.RequestData, CommonData: m.CommonData}
}

func NewResponse(id int, data interface{}) (*Response, error) {
	dataStr, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	response := &Response{
		ResponseData: ResponseData{
			Response: true,
			Ok:       true,
		},
		CommonData: CommonData{
			ID:   id,
			Data: dataStr,
		},
	}
	return response, nil
}

func NewResponseErr(id int, errorCode int, errorReason string) *Response {
	response := &Response{
		ResponseData: ResponseData{
			Response: true,
			Ok:       false,
			ResponseErrData: ResponseErrData{
				ErrorCode:   errorCode,
				ErrorReason: errorReason,
			},
		},
		CommonData: CommonData{
			ID: id,
		},
	}
	return response
}

/*
* Request
{
  request : true,
  id      : 12345678,
  method  : 'chatmessage',
  data    :
  {
    type  : 'text',
    value : 'Hi there!'
  }
}
*/
type Request struct {
	RequestData
	CommonData
}

/*
* Success response
{
	response : true,
	id       : 12345678,
	ok       : true,
	data     :
	{
	  foo : 'lalala'
	}
}
*/
type Response struct {
	ResponseData
	CommonData
}

/*
* Error response
{
  response    : true,
  id          : 12345678,
  ok          : false,
  errorCode   : 123,
  errorReason : 'Something failed'
}
*/
type ResponseError struct {
	ResponseData
	CommonData
}

/*
* Notification
{
  notification : true,
  method       : 'chatmessage',
  data         :
  {
    foo : 'bar'
  }
}
*/
type Notification struct {
	CommonData
	NotificationData
}

// Transcation .
type Transcation struct {
	id     int
	accept AcceptFunc
	reject RejectFunc
	close  func()
	timer  *time.Timer
}
