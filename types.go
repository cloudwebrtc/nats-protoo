package nprotoo

import (
	"encoding/json"
	"time"
)

// AcceptFunc .
type AcceptFunc func(data json.RawMessage)

// RejectFunc .
type RejectFunc func(errorCode int, errorReason string)

// RequestFunc .
type RequestFunc func(request Request, accept AcceptFunc, reject RejectFunc)

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
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Data   json.RawMessage `json:"data"`
}

func (m PeerMsg) ToNotification() Notification {
	return Notification{NotificationData: m.NotificationData, CommonData: m.CommonData}
}

func (m PeerMsg) ToRequest() Request {
	return Request{RequestData: m.RequestData, CommonData: m.CommonData}
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
	Response    bool            `json:"response"`
	ID          int             `json:"id"`
	Ok          bool            `json:"ok"`
	Data        json.RawMessage `json:"data"`
	ErrorCode   int             `json:"errorCode"`
	ErrorReason string          `json:"errorReason"`
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
	Response    bool   `json:"response"`
	ID          int    `json:"id"`
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"errorCode"`
	ErrorReason string `json:"errorReason"`
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
