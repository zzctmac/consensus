package consensus

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"fmt"
)

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

func Test_LocalServer(t *testing.T) {
	Convey("multi", t, func() {
		s := NewLocalServer(10)
		st := 0
		go func() {
			recv := s.Receive()
			for ;; {
				select {
				case p := <-recv:
					p.msg.(*AcceptMessage).N++
					p.c.Send(p)
					st++
					t.Logf("n:%d", p.msg.(*AcceptMessage).N)
				}
			}
		}()
		var wg sync.WaitGroup
		times := 20
		for i := 0 ;i < times;i++ {
			wg.Add(1)
			go func(number int) {
				Convey(fmt.Sprintf("n%d", number), t, func() {
					c := s.Connect()
					c.Send(&Packet{msg: &AcceptMessage{N: int32(number)}})
					p1, err := c.Receive()
					So(err, ShouldBeNil)
					So(p1.msg.(*AcceptMessage).N, ShouldEqual, int32(number+1))
					c.Close()
					wg.Done()
				})
			}(i)
		}
		wg.Wait()
		So(st, ShouldEqual, times)
	})
}

func Test_LocalServer2(t *testing.T) {
	Convey("multi", t, func() {
		s := NewLocalServer(10)
		st := 0
		go func() {
			recv := s.Receive()
			for ;; {
				select {
				case p := <-recv:
					p.msg.(*AcceptMessage).N++
					p.c.Send(p)
					st++
					t.Logf("n:%d", p.msg.(*AcceptMessage).N)
				}
			}
		}()

		c := s.Connect()
		c.Send(&Packet{msg: &AcceptMessage{N: 1}})
		p1, err := c.Receive()
		So(err, ShouldBeNil)
		So(p1.msg.(*AcceptMessage).N, ShouldEqual, 2)

		c.Send(&Packet{msg: &AcceptMessage{N: 2}})
		p2, err := c.Receive()
		So(err, ShouldBeNil)
		So(p2.msg.(*AcceptMessage).N, ShouldEqual, 3)

		So(st, ShouldEqual, 2)
		c.Close()

	})
}