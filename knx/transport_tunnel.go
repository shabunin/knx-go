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
	inbound chan GroupEvent
}

func (t *TransportTunnel) serve() {
	inbound := t.Tunnel.Inbound()
	outbound := t.inbound
	util.Log(inbound, "Started worker")
	defer util.Log(t.inbound, "Worker exited")

	for msg := range inbound {
		var addr cemi.IndividualAddr

		if ind, ok := msg.(*cemi.LDataInd); ok {
			addr = ind.Source
			if app, ok := ind.Data.(*cemi.AppData); ok {
				if app.Command.IsGroupCommand() {
					outbound <- GroupEvent{
						Command:     GroupCommand(app.Command),
						Source:      addr,
						Destination: cemi.GroupAddr(ind.Destination),
						Data:        app.Data,
					}
					continue
				}
			}
		} else if con, ok := msg.(*cemi.LDataCon); ok {
			// confirmation of our own telegram
			addr = cemi.IndividualAddr(con.Destination)
		} else {
			util.Log(t.inbound, "cemi frame contains unknown message code")
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
			t.mu.Lock()
			delete(t.conns, addr)
			t.mu.Unlock()
			continue
		}
		tconn.inbound <- msg
	}

	close(outbound)
}

func (t *TransportTunnel) Dial(addr cemi.IndividualAddr) (*TransportConn, error) {
	t.mu.Lock()
	c, ok := t.conns[addr]
	if ok && !c.Closed() {
		return nil, errors.New("transport connections already exist")
	}
	c = newTransportConn(addr)
	t.conns[addr] = c
	t.mu.Unlock()

	go func() {
		for {
			select {
			case msg, ok := <-c.outbound:
				if !ok {
					t.mu.Lock()
					delete(t.conns, addr)
					t.mu.Unlock()
					return
				}
				t.Tunnel.Send(msg)
			case <-c.ctx.Done():
				t.mu.Lock()
				delete(t.conns, addr)
				t.mu.Unlock()
				return
			}
		}
	}()

	err := c.connect()
	return c, err
}

// NewTransportTunnel creates a new Tunnel with a transport connections support.
func NewTransportTunnel(gatewayAddr string, config TunnelConfig) (tt TransportTunnel, err error) {
	tt = TransportTunnel{
		conns: make(map[cemi.IndividualAddr]*TransportConn),
	}
	tt.Tunnel, err = NewTunnel(gatewayAddr, knxnet.TunnelLayerData, config)
	if err == nil {
		tt.inbound = make(chan GroupEvent)
		go tt.serve()
	}

	return
}

// Send a group communication.
func (tt *TransportTunnel) Send(event GroupEvent) error {
	return tt.Tunnel.Send(&cemi.LDataReq{LData: buildGroupOutbound(event)})
}

// Inbound returns the channel on which group communication can be received.
func (tt *TransportTunnel) Inbound() <-chan GroupEvent {
	return tt.inbound
}
