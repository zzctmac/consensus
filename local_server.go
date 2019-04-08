package consensus

import "sync"

// 通过channel实现全双工的Server&Client　方便测试consensus协议
type LocalServer struct {
	in chan *Packet
	cm map[Client]Client
	buf int
	connectMutex *sync.Mutex
}

func NewLocalServer(buf int)*LocalServer {
	return &LocalServer{in:make(chan *Packet, buf), cm :make(map[Client]Client), buf:buf, connectMutex:&sync.Mutex{}}
}

type LocalClient struct {
	in chan *Packet
	out chan *Packet
}

func (s *LocalServer)Connect() Client {
	s.connectMutex.Lock()
	defer s.connectMutex.Unlock()
	in := make(chan *Packet, s.buf)
	go func() {
		for ;; {
			p, ok  := <-in
			if !ok {
				break
			}
			p.c = s.cm[p.c]
			s.in <- p
		}
	}()

	lc :=  &LocalClient{
		in:in,
		out:make(chan *Packet,s.buf),
	}
	s.cm[lc] = &LocalClient{
		in:lc.out,
		out:nil,
	}


	return lc
}

func (s *LocalServer)Receive() <-chan *Packet {
	return s.in
}

func (c *LocalClient) Send(p *Packet) error {
	p.c = c
	c.in <- p
	return nil
}

func (c *LocalClient) Receive()(p *Packet,err error) {
	p = <- c.out
	p.c = c
	return
}

func (c *LocalClient)Close() error {
	close(c.in)
	close(c.out)
	return nil
}
