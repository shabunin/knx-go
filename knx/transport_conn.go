package knx

import (
	"context"
	"errors"

	"github.com/vapourismo/knx-go/knx/cemi"
)

type TransportConn struct {
	Addr cemi.IndividualAddr

	ctx    context.Context
	cancel context.CancelFunc

	inSeq  uint16
	outSeq uint16

	inbound  chan cemi.Message
	outbound chan cemi.Message
}

func newTransportConn(addr cemi.IndividualAddr) *TransportConn {
	ctx, cancel := context.WithCancel(context.Background())
	t := &TransportConn{
		Addr:     addr,
		ctx:      ctx,
		cancel:   cancel,
		inbound:  make(chan cemi.Message),
		outbound: make(chan cemi.Message),
	}

	go func() {
		<-ctx.Done()
		close(t.inbound)
		close(t.outbound)
	}()

	return t
}

func (t *TransportConn) connect() error {
	// TODO send connect
	go func() {
		// TODO serve <- inbound
	}()
	return nil
}
func (t *TransportConn) disconnect() error {
	// TODO send disconnect request
	return nil
}

func (t *TransportConn) Closed() bool {
	select {
	case <-t.ctx.Done():
		return true
	default:
		return false
	}
}

func (t *TransportConn) Close() error {
	select {
	case <-t.ctx.Done():
		return errors.New("close of closed connection")
	default:
	}

	t.disconnect()
	t.cancel()

	return nil
}
func (t *TransportConn) DeviceDescriptorRead() {}
func (t *TransportConn) PropertyRead()         {}
func (t *TransportConn) PropertyWrite()        {}
func (t *TransportConn) MemoryRead()           {}
func (t *TransportConn) MemoryWrite()          {}
func (t *TransportConn) UserMessage()          {}

func (t *TransportConn) Done() <-chan struct{} {
	return t.ctx.Done()
}
