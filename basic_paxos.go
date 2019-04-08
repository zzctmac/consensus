package consensus

import (
	"log"
	"math"
	"sync"
)

type Message interface {
}

type Packet struct {
	msg Message
	c   Client
}

type Client interface {
	Send(p *Packet) error
	Receive() (*Packet, error)
	Close() error
}

type Server interface {
	Connect() Client
	Receive() <-chan *Packet
}

type PrepareMessage struct {
	N int32
}

type PrepareRespMessage struct {
	AcceptedProposal int32
	AcceptedValue    []byte
}

type AcceptMessage struct {
	N     int32
	value []byte
}

type AcceptRespMessage struct {
	MinProposal int32
}

type BasicPaxos struct {
	Proposers []*Proposer
	Acceptors []*Acceptor
	addMutex  *sync.Mutex
}

func NewBasicPaxos() *BasicPaxos {
	bp := &BasicPaxos{
		addMutex: &sync.Mutex{},
	}
	return bp
}

func (bp *BasicPaxos) AddAcceptor(s Server) *Acceptor {
	bp.addMutex.Lock()
	defer bp.addMutex.Unlock()
	a := &Acceptor{}
	a.Init(bp, s)
	bp.Acceptors = append(bp.Acceptors, a)
	return a
}

func (bp *BasicPaxos) AddProposer(s Server) *Proposer {
	bp.addMutex.Lock()
	defer bp.addMutex.Unlock()
	p := &Proposer{}
	p.Init(int16(len(bp.Proposers)), bp, s)
	bp.Proposers = append(bp.Proposers, p)
	return p
}

func (bp *BasicPaxos) Start() {
	for _, acc := range bp.Acceptors {
		acc.Start()
	}
	for _, p := range bp.Proposers {
		p.connectAcceptors()
	}
}

type Acceptor struct {
	s                Server
	MinProposal      int32
	AcceptedProposal int32
	AcceptedValue    []byte
	bp               *BasicPaxos
}

func (a *Acceptor) Init(bp *BasicPaxos, s Server) {
	a.AcceptedValue = nil
	a.AcceptedProposal = -1
	a.MinProposal = -1
	a.bp = bp
	a.s = s
}
func (a *Acceptor) Start() {

	go func() {
		c := a.s.Receive()
		for {
			select {
			case packet := <-c:
				var resp Message
				switch packet.msg.(type) {
				case *PrepareMessage:
					pm := packet.msg.(*PrepareMessage)
					if pm.N > a.MinProposal {
						a.MinProposal = pm.N
					}
					resp = &PrepareRespMessage{a.AcceptedProposal, a.AcceptedValue}
				case *AcceptMessage:
					am := packet.msg.(*AcceptMessage)
					if am.N >= a.MinProposal {
						a.AcceptedProposal = a.MinProposal
						a.AcceptedValue = am.value
					}
					resp = &AcceptRespMessage{a.MinProposal}
				}
				packet.c.Send(&Packet{msg: resp})
			}
		}
	}()
}

type Proposer struct {
	s               Server
	id              int16
	onumber         int16
	curNumber       int32
	bp              *BasicPaxos
	Value           []byte
	AcceptorClients []Client
	proposeMutex    *sync.Mutex
	receiveChans    []chan int
}

func (p *Proposer) Init(id int16, bp *BasicPaxos, s Server) {
	p.id = id
	p.bp = bp
	p.s = s
	p.proposeMutex = &sync.Mutex{}
}

func (p *Proposer) connectAcceptors() {
	p.AcceptorClients = make([]Client, len(p.bp.Acceptors))
	p.receiveChans = make([]chan int, len(p.bp.Acceptors))
	for i, ac := range p.bp.Acceptors {
		p.AcceptorClients[i] = ac.s.Connect()
		p.receiveChans[i] = make(chan int, 1)
		p.receiveChans[i] <- 1
	}
}

func (p *Proposer) Propose(value []byte) (bool, []byte) {
	p.proposeMutex.Lock()
	defer p.proposeMutex.Unlock()
	// prepare 1
	p.genNumber()
	msg := &PrepareMessage{N: p.curNumber}
	total := len(p.AcceptorClients)
	pc := make(chan *Packet, total)
	maj := int(math.Ceil(float64(total) / 2))
	log.Printf("begin propse maj:%d", maj)
	for i := 0; i < total; i++ {
		go func(i int) {
			<-p.receiveChans[i]
			log.Printf("prepare1 id:%d number:%d", p.id, p.curNumber)
			p.AcceptorClients[i].Send(&Packet{msg: msg})
			packet, _ := p.AcceptorClients[i].Receive()
			p.receiveChans[i] <- 1
			pc <- packet

		}(i)
	}
	// prepare 2
	acceptedValue := value
	acceptedProposal := int32(-1)
	for i := 0; i < total; i++ {
		packet := <-pc
		prm := packet.msg.(*PrepareRespMessage)
		log.Printf("prepare2 id:%d number:%d prm.aP:%d prm.av:%v", p.id, p.curNumber, prm.AcceptedProposal, prm.AcceptedValue)
		if prm.AcceptedValue != nil && prm.AcceptedProposal > acceptedProposal {
			acceptedValue = prm.AcceptedValue
			acceptedProposal = prm.AcceptedProposal
		}

		if i >= maj {
			break
		}
	}

	// accept
	amsg := &AcceptMessage{N: p.curNumber, value: acceptedValue}
	rpc := make(chan *Packet, total)
	for i := 0; i < total; i++ {
		go func(i int) {
			<-p.receiveChans[i]

			log.Printf("accept1 id:%d number:%d, value:%v", p.id, p.curNumber, acceptedValue)
			p.AcceptorClients[i].Send(&Packet{msg: amsg})
			packet, _ := p.AcceptorClients[i].Receive()

			p.receiveChans[i] <- 1
			rpc <- packet

		}(i)
	}

	ac := 0
	for i := 0; i < total; i++ {
		packet := <-rpc
		prm, ok := packet.msg.(*AcceptRespMessage)
		if !ok {
			continue
		}
		log.Printf("accept2 id:%d number:%d, prm.min:%d", p.id, p.curNumber, prm.MinProposal)
		if prm.MinProposal > p.curNumber {
			return false, nil
		}
		ac++
		if ac >= maj {
			break
		}
	}
	p.Value = acceptedValue
	return true, acceptedValue

}

func (p *Proposer) genNumber() {
	p.onumber++
	number := (int32(p.onumber) << 16) | int32(p.id)
	p.curNumber = number
}
