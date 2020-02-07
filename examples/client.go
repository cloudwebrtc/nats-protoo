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
	req.Request("offer", JsonEncode(`{ "sdp": "dummy-sdp"}`),
		func(result map[string]interface{}) {
			logger.Infof("offer success: =>  %s", result)
		},
		func(code int, err string) {
			logger.Infof("offer reject: %d => %s", code, err)
		})

	bc := npc.NewBroadcaster("even1")
	bc.Say("hello", JsonEncode(`{"key": "value"}`))

	select {}
}
