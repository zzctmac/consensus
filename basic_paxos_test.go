package consensus

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"go-common/library/log"
)

func TestProposer_Propose(t *testing.T) {
	Convey("twice", t, func() {
		bp := NewBasicPaxos()
		bp.AddAcceptor(NewLocalServer(10))
		bp.AddAcceptor(NewLocalServer(10))
		bp.AddAcceptor(NewLocalServer(10))

		p1 := bp.AddProposer(nil)
		p2 := bp.AddProposer(nil)
		bp.Start()


		ok ,bs := p1.Propose([]byte("abc"))
		So(ok, ShouldBeTrue)
		So(string(bs), ShouldEqual, "abc")

		ok ,bs = p2.Propose([]byte("abc1"))
		So(ok, ShouldBeTrue)
		So(string(bs), ShouldEqual, "abc")

		ok ,bs = p2.Propose([]byte("abc1"))
		So(ok, ShouldBeTrue)
		So(string(bs), ShouldEqual, "abc")

		ok ,bs = p1.Propose([]byte("abc"))
		So(ok, ShouldBeFalse)

		ok ,bs = p1.Propose([]byte("abcd"))
		So(ok, ShouldBeTrue)
		So(string(bs), ShouldEqual, "abc")

	})

	Convey("parallel", t, func() {
		log.Info("Parallel\n\n")
		bp := NewBasicPaxos()
		bp.AddAcceptor(NewLocalServer(10))
		bp.AddAcceptor(NewLocalServer(10))
		bp.AddAcceptor(NewLocalServer(10))

		p1 := bp.AddProposer(nil)
		p2 := bp.AddProposer(nil)
		bp.Start()

		var ok1 , ok2 bool
		bs1 := []byte("abc")
		bs2 := []byte("abc1")
		var bs1a, bs2a []byte
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			ok1, bs1a = p1.Propose(bs1)
			wg.Done()
		}()

		go func() {
			ok2, bs2a = p2.Propose(bs2)
			wg.Done()
		}()
		wg.Wait()
		t.Logf("bs1a:%s, bs2a:%s", bs1a, bs2a)

		var r []byte
		if ok1 || ok2 {
			if ok1 {
				if string(bs1a) == string(bs1) {
					r = bs1a
				} else {
					r = bs2a
				}
			} else {
				r = bs2a
			}
			ok ,qr := p1.Propose([]byte("qqq"))
			t.Logf("qr:%s r :%s bs1:%s bs2:%s ok1:%v ok2:%v bs1a:%s bs2a:%s", qr, r, bs1, bs2, ok1, ok2, bs1a, bs2a)
			So(ok, ShouldBeTrue)
			So(string(r), ShouldEqual, string(qr))

		} else {
			t.Logf("both failed")
		}


	})
}
