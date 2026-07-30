package server

import (
	"fmt"
	"sync"
)

const (
	magicLinkDeliveryQueueCapacity = 32
	magicLinkDeliveryWorkers       = 1
)

type magicLinkDeliveryJob struct {
	mailer  mailer
	to      string
	link    string
	address string
}

// magicLinkDeliveryQueue keeps the public login request independent from SMTP
// latency without spawning one unbounded goroutine per request. Enqueue is
// deliberately non-blocking: admission failure is handled like every other
// login-delivery failure and never changes the public response.
type magicLinkDeliveryQueue struct {
	mu        sync.RWMutex
	closeOnce sync.Once
	jobs      chan magicLinkDeliveryJob
	closed    bool
	workers   sync.WaitGroup
}

func newMagicLinkDeliveryQueue(capacity int, workers int) *magicLinkDeliveryQueue {
	if capacity < 1 {
		capacity = 1
	}
	if workers < 1 {
		workers = 1
	}
	queue := &magicLinkDeliveryQueue{
		jobs: make(chan magicLinkDeliveryJob, capacity),
	}
	queue.workers.Add(workers)
	for range workers {
		go queue.run()
	}
	return queue
}

func (q *magicLinkDeliveryQueue) enqueue(job magicLinkDeliveryJob) bool {
	if q == nil || job.mailer == nil {
		return false
	}
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return false
	}
	select {
	case q.jobs <- job:
		return true
	default:
		return false
	}
}

func (q *magicLinkDeliveryQueue) run() {
	defer q.workers.Done()
	for job := range q.jobs {
		deliverMagicLink(job)
	}
}

func deliverMagicLink(job magicLinkDeliveryJob) {
	defer func() {
		if recovered := recover(); recovered != nil {
			// Never include the panic value: a transport implementation could
			// embed recipient or message data in it.
			logWarn("magic link delivery panicked",
				"panic_type", fmt.Sprintf("%T", recovered),
			)
		}
	}()
	if err := job.mailer.SendMagicLink(job.to, job.link, job.address); err != nil {
		// The SMTP error text is intentionally omitted. Relay responses may
		// contain the recipient and must not turn logs into a PII side channel.
		logWarn("magic link delivery failed",
			"error_type", fmt.Sprintf("%T", err),
		)
	}
}

// close drains all jobs accepted before shutdown and waits for the fixed worker
// set. main closes the HTTP server before app.Close, so no new login requests
// can race in after this lifecycle boundary.
func (q *magicLinkDeliveryQueue) close() {
	if q == nil {
		return
	}
	q.closeOnce.Do(func() {
		q.mu.Lock()
		q.closed = true
		close(q.jobs)
		q.mu.Unlock()
		q.workers.Wait()
	})
}

func (a *app) enqueueMagicLinkDelivery(job magicLinkDeliveryJob) bool {
	a.magicLinkDeliveryMu.Lock()
	if a.magicLinkDeliveryClosed {
		a.magicLinkDeliveryMu.Unlock()
		return false
	}
	if a.magicLinkDelivery == nil {
		a.magicLinkDelivery = newMagicLinkDeliveryQueue(
			magicLinkDeliveryQueueCapacity,
			magicLinkDeliveryWorkers,
		)
	}
	queue := a.magicLinkDelivery
	a.magicLinkDeliveryMu.Unlock()
	return queue.enqueue(job)
}

func (a *app) closeMagicLinkDelivery() {
	if a == nil {
		return
	}
	a.magicLinkDeliveryMu.Lock()
	a.magicLinkDeliveryClosed = true
	queue := a.magicLinkDelivery
	a.magicLinkDeliveryMu.Unlock()
	queue.close()
}
