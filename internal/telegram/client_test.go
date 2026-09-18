package telegram

import (
	"sync/atomic"
	"testing"
)

func TestWorkerPoolAcquisition(t *testing.T) {
	w1 := &ClientWorker{
		ID:        1001,
		Username:  "BotOne",
		Ready:     true,
		Workload:  0,
	}
	w2 := &ClientWorker{
		ID:        1002,
		Username:  "BotTwo",
		Ready:     true,
		Workload:  0,
	}

	svc := &Service{
		primaryWorker: w1,
		workers:       []*ClientWorker{w1, w2},
	}

	// 1. First acquisition should pick w1 (or w2 with load 0)
	acquired1 := svc.AcquireWorker()
	if acquired1 == nil {
		t.Fatal("expected worker to be acquired")
	}
	if atomic.LoadInt64(&acquired1.Workload) != 1 {
		t.Errorf("expected workload 1, got %d", acquired1.Workload)
	}

	// 2. Next acquisition should pick the worker with load 0 (the other one)
	acquired2 := svc.AcquireWorker()
	if acquired2 == nil {
		t.Fatal("expected second worker to be acquired")
	}
	if acquired2 == acquired1 {
		t.Errorf("expected least loaded worker to be picked, but got the same worker")
	}

	// 3. Acquire by specific Bot ID
	bot2 := svc.AcquireWorkerForBot("1002")
	if bot2 == nil || bot2.ID != 1002 {
		t.Errorf("expected worker with ID 1002, got %+v", bot2)
	}

	// 4. Release worker decrements load
	svc.ReleaseWorker(bot2)
	if atomic.LoadInt64(&bot2.Workload) != 1 {
		t.Errorf("expected workload to be 1 after release, got %d", bot2.Workload)
	}

	// Status string
	status := svc.StatusString()
	if status != "connected (@BotOne, @BotTwo)" {
		t.Errorf("expected 'connected (@BotOne, @BotTwo)', got %q", status)
	}
}
