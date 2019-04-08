package consensus

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
