package nprotoo

import "encoding/json"

// Error .
type Error struct {
	Code   int
	Reason string
}

// Future .
type Future struct {
	c      chan struct{}
	result json.RawMessage
	err    *Error
}

// NewFuture .
func NewFuture() *Future {
	future := Future{
		c:   make(chan struct{}, 1),
		err: nil,
	}
	return &future
}

// Await .
func (future *Future) Await() (json.RawMessage, *Error) {
	<-future.c
	return future.result, future.err
}

// Then .
func (future *Future) Then(resolve func(result json.RawMessage), reject func(err *Error)) {
	go func() {
		<-future.c
		if future.err != nil {
			reject(future.err)
		} else {
			resolve(future.result)
		}
	}()
}

func (future *Future) resolve(result json.RawMessage) {
	future.result = result
	close(future.c)
}

func (future *Future) reject(err *Error) {
	future.err = err
	close(future.c)
}
