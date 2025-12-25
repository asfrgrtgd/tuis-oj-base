package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"tuis-oj-prototype/core"
)

func main() {
	cfg := core.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logCloser, err := core.SetupLogging(cfg, "worker.log")
	if err != nil {
		log.Fatalf("failed to setup logging: %v", err)
	}
	defer logCloser.Close()

	db, err := core.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	redisClient, err := core.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer redisClient.Close()

	queue := core.NewRedisQueue(redisClient)
	repo := core.NewPgSubmissionRepository(db)
	problemRepo := core.NewPgProblemRepository(db)
	judge := core.NewHTTPJudgeClient(cfg.GoJudgeURL)
	processor := core.NewWorkerProcessor(repo, problemRepo, judge, cfg.CompileTimeLimitMs)
	concurrency := cfg.WorkerConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	workerID := core.NewWorkerID()
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	username := "unknown"
	if currentUser != nil && currentUser.Username != "" {
		username = currentUser.Username
	}
	log.Printf("worker started. id=%s concurrency=%d queue=%s judge=%s user=%s", workerID, concurrency, core.PendingQueueKey, cfg.GoJudgeURL, username)

	const pendingKey = core.PendingQueueKey
	const processingKey = core.ProcessingQueueKey
	visibility := core.DefaultVisibilityTimeout
	reclaimInterval := 15 * time.Second
	const maxRetries = 3

	state := core.NewHeartbeatState(workerID, hostname, concurrency)
	go state.Start(ctx, redisClient)

	// requeue expired in-flight jobs periodically
	go func() {
		ticker := time.NewTicker(reclaimInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if jobs, err := queue.RequeueExpired(ctx, processingKey, pendingKey, time.Now()); err != nil {
					log.Printf("[reclaimer] requeue expired error: %v", err)
				} else if len(jobs) > 0 {
					for _, job := range jobs {
						if id, err := strconv.ParseInt(job, 10, 64); err == nil {
							_ = repo.MarkStatus(ctx, id, "pending")
							_, _ = repo.IncrementRetry(ctx, id)
						}
					}
					log.Printf("[reclaimer] requeued %d expired jobs", len(jobs))
				}
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				job, err := queue.Reserve(ctx, pendingKey, processingKey, visibility)
				if err != nil {
					if errors.Is(err, redis.Nil) {
						// Queue is empty, wait before retrying to avoid CPU spinning
						select {
						case <-ctx.Done():
							return
						case <-time.After(100 * time.Millisecond):
							continue
						}
					}
					// context canceled -> exit
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						return
					}
					log.Printf("[worker %d] dequeue error: %v", workerID, err)
					time.Sleep(time.Second)
					continue
				}

				log.Printf("[worker %d] received job %s", workerID, job)
				state.JobStarted(job)

				verdict, procErr := processor.Process(ctx, job)
				if procErr != nil {
					id, parseErr := strconv.ParseInt(job, 10, 64)
					if parseErr != nil {
						log.Printf("[worker %d] parse job id error for %s: %v", workerID, job, parseErr)
						_ = queue.Ack(ctx, processingKey, job)
						continue
					}

					if errors.Is(procErr, core.ErrSubmissionNotPending) {
						log.Printf("[worker %d] skip job %s: already processed", workerID, job)
						_ = queue.Ack(ctx, processingKey, job)
						continue
					}

					newRetry, incErr := repo.IncrementRetry(ctx, id)
					if incErr != nil {
						log.Printf("[worker %d] increment retry failed for job %s: %v", workerID, job, incErr)
					}

					if newRetry <= maxRetries {
						_ = repo.MarkStatus(ctx, id, "pending")
						if err := queue.Enqueue(ctx, pendingKey, job); err != nil {
							log.Printf("[worker %d] re-enqueue job %s failed: %v", workerID, job, err)
						} else {
							log.Printf("[worker %d] job %s retried (retry_count=%d)", workerID, job, newRetry)
						}
					} else {
						errMsg := procErr.Error()
						res := core.SubmissionResult{
							SubmissionID: id,
							Verdict:      "SE",
							ErrorMessage: &errMsg,
						}
						if saveErr := repo.SaveResult(ctx, res, "failed"); saveErr != nil {
							log.Printf("[worker %d] final fail save result job %s: %v", workerID, job, saveErr)
						}
						log.Printf("[worker %d] job %s failed after retries (retry_count=%d)", workerID, job, newRetry)
					}
				} else if verdict != "AC" {
					log.Printf("[worker %d] job %s finished with verdict=%s", workerID, job, verdict)
				}

				if err := queue.Ack(ctx, processingKey, job); err != nil {
					log.Printf("[worker %d] ack failed for job %s: %v", workerID, job, err)
				}
				state.JobFinished(job, procErr)
			}
		}(i + 1)
	}

	wg.Wait()
}
