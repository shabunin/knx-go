package knx

import (
	"errors"
	"sync"

	"github.com/vapourismo/knx-go/knx/cemi"
	"github.com/vapourismo/knx-go/knx/knxnet"
	"github.com/vapourismo/knx-go/knx/util"
)

type TransportTunnel struct {
	*Tunnel
	mu      sync.RWMutex
	conns   map[cemi.IndividualAddr]*TransportConn
	inbound chan cemi.Message
}

func (t *TransportTunnel) serve() {
	inbound := t.Tunnel.Inbound()
	outbound := t.inbound
	util.Log(inbound, "Started worker")
	defer util.Log(t.inbound, "Worker exited")

	for msg := range inbound {
		var isGroup bool
		var addr cemi.IndividualAddr

		if ind, ok := msg.(*cemi.LDataInd); ok {
			isGroup = ind.Control2.IsGroupAddr()
			addr = ind.Source
		} else if con, ok := msg.(*cemi.LDataCon); ok {
			// confirmation of our own telegram
			isGroup = con.Control2.IsGroupAddr()
			addr = cemi.IndividualAddr(ind.Destination)
		} else {
			util.Log(t.inbound, "cemi frame does not belong to transport connection")
			outbound <- msg
			continue
		}

		if isGroup {
			util.Log(t.inbound, "LData frame target is Group Address")
			outbound <- msg
			continue
		}
		t.mu.RLock()
		tconn, ok := t.conns[addr]
		t.mu.RUnlock()
		if !ok {
			util.Log(t.inbound, "transport connection not found")
			continue
		}
		if tconn.Closed() {
			util.Log(t.inbound, "transport connection found but closed")
			// TODO delete
			continue
		}
		tconn.inbound <- msg
	}

	close(outbound)
}

func (t *TransportTunnel) Dial(addr cemi.IndividualAddr) (*TransportConn, error) {
	t.mu.Lock()
	_, ok := t.conns[addr]
	if ok {
		return nil, errors.New("transport connections already exist")
	}
	c := newTransportConn(addr)
	t.conns[addr] = c
	t.mu.Unlock()
	go func() {
		<-c.Done()
		t.mu.Lock()
		defer t.mu.Unlock()
		delete(t.conns, addr)
	}()
	return c, nil
}

// NewTransportTunnel creates a new Tunnel with a transport connections support.
func NewTransportTunnel(gatewayAddr string, config TunnelConfig) (tt TransportTunnel, err error) {
	tt.Tunnel, err = NewTunnel(gatewayAddr, knxnet.TunnelLayerData, config)

	if err == nil {
		tt.inbound = make(chan cemi.Message)
		go tt.serve()
	}

	return
}

// Send a group communication.
func (gt *GroupTunnel) Send(event GroupEvent) error {
	return gt.Tunnel.Send(&cemi.LDataReq{LData: buildGroupOutbound(event)})
}

// Inbound returns the channel on which group communication can be received.
func (gt *GroupTunnel) Inbound() <-chan GroupEvent {
	return gt.inbound
}
