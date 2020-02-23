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

	req.Request("offer", JsonEncode(`{ "sdp": "dummy-sdp1"}`),
		func(result map[string]interface{}) {
			logger.Infof("offer success: =>  %s", result)
		},
		func(code int, err string) {
			logger.Warnf("offer reject: %d => %s", code, err)
		})

	req.RequestAsFuture("offer", JsonEncode(`{ "sdp": "dummy-sdp2"}`))

	result, err := req.RequestAsFuture("offer", JsonEncode(`{ "sdp": "dummy-sdp3"}`)).Await()
	if err != nil {
		logger.Warnf("offer reject: %d => %s", err.Code, err.Reason)
	} else {
		logger.Infof("offer success: =>  %s", result)
	}

	bc := npc.NewBroadcaster("even1")
	bc.Say("hello", JsonEncode(`{"key": "value"}`))

	select {}
}
