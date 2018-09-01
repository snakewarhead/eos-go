package p2p

import (
	"fmt"

	"time"

	"sync"

	"github.com/snakewarhead/eos-go"
)

type Proxy struct {
	Peer1                       *Peer
	Peer2                       *Peer
	handlers                    []Handler
	handlersLock                sync.Mutex
	waitingOriginHandShake      bool
	waitingDestinationHandShake bool
}

func NewProxy(peer1 *Peer, peer2 *Peer) *Proxy {
	return &Proxy{
		Peer1: peer1,
		Peer2: peer2,
	}
}

func (p *Proxy) RegisterHandler(handler Handler) {
	p.handlersLock.Lock()
	defer p.handlersLock.Unlock()

	p.handlers = append(p.handlers, handler)
}

func (p *Proxy) read(sender *Peer, receiver *Peer, errChannel chan error) {
	for {
		packet, err := sender.Read()
		if err != nil {
			errChannel <- fmt.Errorf("read message from %s: %s", sender.Address, err)
		}
		err = p.handle(packet, sender, receiver)
		if err != nil {
			errChannel <- err
		}
	}
}

func (p *Proxy) handle(packet *eos.Packet, sender *Peer, receiver *Peer) error {

	_, err := receiver.Write(packet.Raw)
	if err != nil {
		return fmt.Errorf("handleDefault: %s", err)
	}

	switch m := packet.P2PMessage.(type) {
	case *eos.GoAwayMessage:
		return fmt.Errorf("handling message: go away: reason [%d]", m.Reason)
	}

	envelope := NewEnvelope(sender, receiver, packet)

	p.handlersLock.Lock()
	for _, handle := range p.handlers {
		handle.Handle(envelope)
	}
	p.handlersLock.Unlock()

	return nil
}

func triggerHandshake(peer *Peer) error {
	dummyHandshakeInfo := &HandshakeInfo{
		HeadBlockID:   make([]byte, 32),
		HeadBlockNum:  0,
		HeadBlockTime: time.Now(),
	}
	fmt.Println("Sending dummy handshake to: ", peer.Address)
	// Process will resume in handle()
	return peer.SendHandshake(dummyHandshakeInfo)
}

func (p *Proxy) Start() error {

	errorChannel := make(chan error)

	peer1ReadyChannel := p.Peer1.Init(errorChannel)
	peer2ReadyChannel := p.Peer2.Init(errorChannel)

	peer1Ready := false
	peer2Ready := false
	for {

		select {
		case <-peer1ReadyChannel:
			peer1Ready = true
			if p.Peer1.mockHandshake {
				err := triggerHandshake(p.Peer1)
				if err != nil {
					return err
				}
			}
		case <-peer2ReadyChannel:
			peer2Ready = true
			if p.Peer2.mockHandshake {
				err := triggerHandshake(p.Peer2)
				if err != nil {
					return err
				}
			}
		case err := <-errorChannel:
			return err
		}

		if peer1Ready && peer2Ready {
			go p.read(p.Peer1, p.Peer2, errorChannel)
			go p.read(p.Peer2, p.Peer1, errorChannel)

		}
	}
}
