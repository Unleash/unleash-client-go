package unleash_client_go

import "io"

type errorEmitter interface {
	io.Closer
	warn(error)
	err(error)
	Warnings() <-chan error
	Errors() <-chan error
	Forward(errorEmitter)
}

type errorEmitterImpl struct {
	warnings chan error
	errors   chan error
	close    chan bool
}

func newErrorEmitter() *errorEmitterImpl {
	return &errorEmitterImpl{
		warnings: make(chan error, 5),
		errors:   make(chan error, 5),
		close:    make(chan bool),
	}
}

func (e errorEmitterImpl) warn(warning error) {
	e.warnings <- warning
}

func (e errorEmitterImpl) err(err error) {
	e.errors <- err
}

func (e errorEmitterImpl) Close() error {
	e.close <- true
	return nil
}

func (e errorEmitterImpl) Warnings() <-chan error {
	return e.warnings
}

func (e errorEmitterImpl) Errors() <-chan error {
	return e.errors
}

func (e errorEmitterImpl) Forward(to errorEmitter) {
	go func() {
		for {
			select {
			case w := <-e.Warnings():
				to.warn(w)
			case err := <-e.Errors():
				to.err(err)
			case <-e.close:
				break
			}
		}
	}()
}
