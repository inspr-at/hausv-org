package server

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	magicLinkDeliveryQueueCapacity       = 32
	magicLinkDeliveryWorkers             = 1
	magicLinkDeliveryShutdownTimeout     = 5 * time.Second
	magicLinkDeliveryCancellationTimeout = time.Second
)

type magicLinkDeliveryJob struct {
	mailer     mailer
	to         string
	link       string
	address    string
	invalidate func()
}

type contextualMagicLinkMailer interface {
	SendMagicLinkContext(context.Context, string, string, string) error
}

// magicLinkDeliveryQueue keeps the public login request independent from SMTP
// latency without spawning one unbounded goroutine per request. Enqueue is
// deliberately non-blocking: admission failure is handled like every other
// login-delivery failure and never changes the public response.
type magicLinkDeliveryQueue struct {
	mu            sync.RWMutex
	closeOnce     sync.Once
	jobs          chan magicLinkDeliveryJob
	closed        bool
	ctx           context.Context
	cancel        context.CancelFunc
	workers       sync.WaitGroup
	workersDone   chan struct{}
	closeFinished chan struct{}
	drained       bool
}

func newMagicLinkDeliveryQueue(capacity int, workers int) *magicLinkDeliveryQueue {
	if capacity < 1 {
		capacity = 1
	}
	if workers < 1 {
		workers = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	queue := &magicLinkDeliveryQueue{
		jobs:          make(chan magicLinkDeliveryJob, capacity),
		ctx:           ctx,
		cancel:        cancel,
		workersDone:   make(chan struct{}),
		closeFinished: make(chan struct{}),
	}
	queue.workers.Add(workers)
	for range workers {
		go queue.run()
	}
	go func() {
		queue.workers.Wait()
		close(queue.workersDone)
	}()
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
		if q.ctx.Err() != nil {
			job.invalidateToken()
			continue
		}
		deliverMagicLink(q.ctx, job)
	}
}

func deliverMagicLink(ctx context.Context, job magicLinkDeliveryJob) {
	defer func() {
		if recovered := recover(); recovered != nil {
			job.invalidateToken()
			// Never include the panic value: a transport implementation could
			// embed recipient or message data in it.
			logWarn("magic link delivery panicked",
				"panic_type", fmt.Sprintf("%T", recovered),
			)
		}
	}()
	err := sendMagicLinkWithContext(ctx, job.mailer, job.to, job.link, job.address)
	if err != nil {
		job.invalidateToken()
		// The SMTP error text is intentionally omitted. Relay responses may
		// contain the recipient and must not turn logs into a PII side channel.
		logWarn("magic link delivery failed",
			"error_type", fmt.Sprintf("%T", err),
		)
	}
}

func sendMagicLinkWithContext(ctx context.Context, sender mailer, to string, link string, address string) error {
	if contextual, ok := sender.(contextualMagicLinkMailer); ok {
		return contextual.SendMagicLinkContext(ctx, to, link, address)
	}
	return sender.SendMagicLink(to, link, address)
}

func (job magicLinkDeliveryJob) invalidateToken() {
	if job.invalidate != nil {
		job.invalidate()
	}
}

// close first gives accepted jobs a short, fixed window to finish. At the
// boundary it cancels the active context, invalidates queued tokens, and waits
// only for a second fixed cancellation window. A non-cooperative transport can
// therefore retain at most the fixed worker set until process exit, never hold
// app.Close indefinitely or create one goroutine per request.
func (q *magicLinkDeliveryQueue) close(drainTimeout time.Duration, cancellationTimeout time.Duration) bool {
	if q == nil {
		return true
	}
	q.closeOnce.Do(func() {
		q.mu.Lock()
		q.closed = true
		close(q.jobs)
		q.mu.Unlock()

		drainTimer := time.NewTimer(nonNegativeDuration(drainTimeout))
		defer drainTimer.Stop()
		select {
		case <-q.workersDone:
			q.drained = true
			q.cancel()
		case <-drainTimer.C:
			q.cancel()
			q.invalidatePending()
			cancelTimer := time.NewTimer(nonNegativeDuration(cancellationTimeout))
			defer cancelTimer.Stop()
			select {
			case <-q.workersDone:
			case <-cancelTimer.C:
			}
		}
		close(q.closeFinished)
	})
	<-q.closeFinished
	return q.drained
}

func (q *magicLinkDeliveryQueue) invalidatePending() {
	for job := range q.jobs {
		job.invalidateToken()
	}
}

func nonNegativeDuration(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
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
	if !queue.close(magicLinkDeliveryShutdownTimeout, magicLinkDeliveryCancellationTimeout) {
		logWarn("magic link delivery shutdown deadline reached")
	}
}
