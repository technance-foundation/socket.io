package queue

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestQueue_EnqueueNil(t *testing.T) {
	q := New()
	defer q.Close()

	// Enqueue nil should be silently ignored
	q.Enqueue(nil)

	// Enqueue a real task to verify queue still works
	done := make(chan struct{})
	q.Enqueue(func() { close(done) })

	select {
	case <-done:
		// success
	case <-time.After(time.Second):
		t.Fatal("Queue stuck after enqueueing nil")
	}
}

func TestQueue_PanicRecovery(t *testing.T) {
	q := New()
	defer q.Close()

	// Enqueue a panicking task
	q.Enqueue(func() {
		panic("test panic")
	})

	// Enqueue a normal task after the panic to verify queue continues
	done := make(chan struct{})
	q.Enqueue(func() {
		close(done)
	})

	select {
	case <-done:
		// Queue recovered from panic and processed next task
	case <-time.After(2 * time.Second):
		t.Fatal("Queue did not recover from panic")
	}
}

func TestQueue_TryClose(t *testing.T) {
	q := New()

	var executed atomic.Int32
	for range 10 {
		q.Enqueue(func() {
			executed.Add(1)
		})
	}

	// TryClose should not block
	q.TryClose()

	// After TryClose, Enqueue should be a no-op
	q.Enqueue(func() {
		executed.Add(100) // should not execute
	})

	// Wait a bit for queue to wind down
	time.Sleep(100 * time.Millisecond)

	if got := executed.Load(); got > 10 {
		t.Errorf("Task executed after TryClose: count=%d", got)
	}
}

func TestQueue_TryCloseAndEnqueue(t *testing.T) {
	q := New()
	q.TryClose()

	// Enqueue after TryClose should not execute
	executed := false
	q.Enqueue(func() {
		executed = true
	})

	time.Sleep(50 * time.Millisecond)
	if executed {
		t.Error("Task should not execute after TryClose")
	}
}

func TestQueue_TryCloseDropsPendingTasks(t *testing.T) {
	q := New()

	started := make(chan struct{})
	release := make(chan struct{})
	q.Enqueue(func() {
		close(started)
		<-release
	})
	<-started

	var executed atomic.Int32
	for range 100 {
		q.Enqueue(func() {
			executed.Add(1)
		})
	}

	q.TryClose()
	close(release)

	select {
	case <-q.done:
	case <-time.After(time.Second):
		t.Fatal("Queue did not stop after TryClose")
	}

	if got := executed.Load(); got != 0 {
		t.Fatalf("expected TryClose to drop pending tasks, got %d executions", got)
	}
}

func TestQueue_ReleasesLargeBackingArrayWhenIdle(t *testing.T) {
	q := New()
	defer q.Close()

	started := make(chan struct{})
	release := make(chan struct{})
	q.Enqueue(func() {
		close(started)
		<-release
	})
	<-started

	for range 2048 {
		q.Enqueue(func() {})
	}

	done := make(chan struct{})
	q.Enqueue(func() {
		close(done)
	})

	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Queue did not drain")
	}

	q.mu.Lock()
	tasksLen := len(q.tasks)
	tasksCap := cap(q.tasks)
	q.mu.Unlock()

	if tasksLen != 0 {
		t.Fatalf("expected empty queue, got len=%d", tasksLen)
	}
	if tasksCap != 0 {
		t.Fatalf("expected large idle backing array to be released, got cap=%d", tasksCap)
	}
}

func TestQueue_SizeEmpty(t *testing.T) {
	q := New()
	defer q.Close()

	if got := q.Size(); got != 0 {
		t.Errorf("Empty queue Size() = %d, want 0", got)
	}
}

func TestQueue_SizeAfterConsumption(t *testing.T) {
	q := New()
	defer q.Close()

	done := make(chan struct{})
	q.Enqueue(func() {
		close(done)
	})

	<-done
	// After task is consumed, size should be 0
	// Small delay to allow queue internal state to settle
	time.Sleep(10 * time.Millisecond)
	if got := q.Size(); got != 0 {
		t.Errorf("Size() after consumption = %d, want 0", got)
	}
}

func TestQueue_MultiplePanics(t *testing.T) {
	q := New()
	defer q.Close()

	// Multiple panics shouldn't break the queue
	for i := range 5 {
		q.Enqueue(func() {
			panic("panic " + string(rune('0'+i)))
		})
	}

	done := make(chan struct{})
	q.Enqueue(func() {
		close(done)
	})

	select {
	case <-done:
		// Queue survived multiple panics
	case <-time.After(2 * time.Second):
		t.Fatal("Queue did not survive multiple panics")
	}
}
