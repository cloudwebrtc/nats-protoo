# protocol

## Listener

``` go
nprotoo := nprotoo.NewNantsProtoo("nats://127.0.0.1:4222")

nprotoo.OnRequest("node-controller-channel", func(request map[string]interface{}, accept AcceptFunc, reject RejectFunc) {

    method := request["method"].(string)
    data := request["data"].(map[string]interface{})

    accept(`{}`)
    reject(404,"Not found")
})

broadcaster := nprotoo.NewBroadcaster("node-controller-channel")
broadcaster.Say(`{key: value}`)
```

## Client

``` go
nprotoo := nprotoo.NewNantsProtoo("nats://127.0.0.1:4222")

nprotoo.OnBroadcast("node-controller-channel", func(data interface{}) {
 log.Printf("Got Broadcast => %v", data)
})

requestor := nprotoo.NewRequestor("node-controller-channel")

requestor.Request("offer",  `{ "sdp": "dummy-sdp"}`,
  func(result map[string]interface{}) {
       logger.Infof("offer success: =>  %s", result)
  },
  func(code int, err string) {
       logger.Infof("offer reject: %d => %s", code, err)
  }
)
```
