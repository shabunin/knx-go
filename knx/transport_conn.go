package knx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vapourismo/knx-go/knx/cemi"
)

type TransportConn struct {
	Addr cemi.IndividualAddr

	closed bool

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

	return t
}

func (t *TransportConn) connect() error {
	msg := cemi.LDataReq{
		LData: cemi.LData{
			Control1: cemi.Control1NoRepeat | cemi.Control1NoSysBroadcast |
				cemi.Control1WantAck | cemi.Control1Prio(cemi.PrioSystem),
			Control2: cemi.Control2Hops(6),
		},
	}
	msg.LData.Destination = uint16(t.Addr)
	msg.LData.Data = &cemi.AppData{
		Numbered:  false,
		SeqNumber: 0,
		Data:      []byte{},
	}
	t.outbound <- &msg

	select {
	case income := <-t.inbound:
		fmt.Printf("income: %s, %v\n", income.MessageCode().String(), income)
		if con, ok := income.(*cemi.LDataCon); ok {
			if (con.Control1 & cemi.Control1HasError) == 0x1 {
				return errors.New("connection error")
			}
		}
		return nil
	case <-time.After(time.Second * 3): // TODO consult spec
		t.disconnect()
	}

	return nil
}
func (t *TransportConn) disconnect() error {
	// cancel context
	t.cancel()

	// TODO send disconnect request but don't expect answer
	close(t.inbound)
	close(t.outbound)
	t.closed = true

	return nil
}

func (t *TransportConn) Closed() bool {
	return t.closed
}

func (t *TransportConn) Close() error {
	if t.closed {
		return errors.New("close of closed connection")
	}
	t.disconnect()

	return nil
}
func (t *TransportConn) DeviceDescriptorRead() {}
func (t *TransportConn) PropertyRead()         {}
func (t *TransportConn) PropertyWrite()        {}
func (t *TransportConn) MemoryRead()           {}
func (t *TransportConn) MemoryWrite()          {}
func (t *TransportConn) UserMessage()          {}
