package nprotoo

// Error .
type Error struct {
	Code  int
	Reason string
}

// Future .
type Future struct {
	c chan struct{}
	result map[string]interface{}
	err *Error
}

// NewFuture .
func NewFuture() *Future {
	future := Future{
		c: make(chan struct{}, 1),
		result: make(map[string]interface{}),
		err: nil,
	}
	return &future
}

// Await .
func (future *Future) Await() (map[string]interface{}, *Error) {
	<-future.c
	return future.result, future.err
}

func (future *Future) resolve(result map[string]interface{}){
	future.result = result
	close(future.c)
}

func (future *Future) reject(err *Error){
	future.err = err
	close(future.c)
}
