package agent

import (
	"reflect"
	"sync"
	"testing"
)

func TestSteeringRecallOrderingAndAcceptanceRace(t *testing.T) {
	s := &Steering{}
	s.Push("first")
	s.Push("second")
	if text, ok := s.Recall(); !ok || text != "second" {
		t.Fatal("recall must return newest pending instruction")
	}
	s.Push("edited second")
	if got := s.Take(); !reflect.DeepEqual(got, []string{"first", "edited second"}) {
		t.Fatal(got)
	}
	if _, ok := s.Recall(); ok {
		t.Fatal("accepted instruction recalled")
	}
	for n := 0; n < 100; n++ {
		queue := &Steering{}
		queue.Push("unique")
		var taken []string
		var recalled bool
		var group sync.WaitGroup
		group.Add(2)
		go func() { defer group.Done(); taken = queue.Take() }()
		go func() { defer group.Done(); _, recalled = queue.Recall() }()
		group.Wait()
		if (len(taken) == 1) == recalled {
			t.Fatal("instruction was lost or both recalled and accepted")
		}
	}
}
