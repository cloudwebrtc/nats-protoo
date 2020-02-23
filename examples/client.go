package main

import (
	"encoding/json"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/cloudwebrtc/nats-protoo/logger"
)

func JsonEncode(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		panic(err)
	}
	return data
}

func main() {
	logger.Init("debug")
	npc := nprotoo.NewNatsProtoo(nprotoo.DefaultNatsURL)
	req := npc.NewRequestor("channel1")

	req.AsyncRequest("offer", JsonEncode(`{ "sdp": "dummy-sdp1"}`)).Then(
		func(result map[string]interface{}) {
			logger.Infof("AsyncRequest.Then: offer success: =>  %s", result)
		},
		func(err *nprotoo.Error) {
			logger.Warnf("AsyncRequest.Then: offer reject: %d => %s", err.Code, err.Reason)
		})

	result, err := req.SyncRequest("offer", JsonEncode(`{ "sdp": "dummy-sdp3"}`))
	if err != nil {
		logger.Warnf("offer reject: %d => %s", err.Code, err.Reason)
	} else {
		logger.Infof("offer success: =>  %s", result)
	}

	req.AsyncRequest("offer", JsonEncode(`{ "sdp": "dummy-sdp2"}`))

	bc := npc.NewBroadcaster("even1")
	bc.Say("hello", JsonEncode(`{"key": "value"}`))

	select {}
}
