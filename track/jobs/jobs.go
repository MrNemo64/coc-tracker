package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type JobState string

var (
	JobStatePending = "pending"
	JobStateQueued  = "queued"
	JobStateRunning = "running"
)

type JobRunContext interface {
	GetDB() *sqlx.DB
	Get(ctx context.Context, url string) (response *http.Response, cacheHit bool, err error)
}

type Job interface {
	Run(JobRunContext, context.Context) (*JobFinishInformation, error)
	Serialize(*sqlx.DB) error
}

type ScheduleInformation struct {
	At   time.Time
	Data any
}

type JobFinishInformation struct {
	Successfull bool
	Reschedule  *ScheduleInformation
}

type JobProvider interface {
	Deserialize(string) (Job, error)
	Save(*sqlx.Tx, *ScheduleInformation) error
	CheckJobsTable(*sqlx.DB) error
	JobName() string
}

type RegisteredJobs struct {
	providers map[string]JobProvider
}

func NewJobQueue() *RegisteredJobs {
	return &RegisteredJobs{
		providers: make(map[string]JobProvider),
	}
}

func (q *RegisteredJobs) RegisterJobKind(provider JobProvider) {
	q.providers[provider.JobName()] = provider
}

func (q *RegisteredJobs) CheckJobs(db *sqlx.DB) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(q.providers))

	for name, provider := range q.providers {
		wg.Add(1)
		go func(name string, provider JobProvider) {
			defer wg.Done()
			if err := provider.CheckJobsTable(db); err != nil {
				errCh <- fmt.Errorf("error in provider %s: %w", name, err)
			}
		}(name, provider)
	}

	wg.Wait()
	close(errCh)

	var allErrors []error
	for err := range errCh {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("multiple errors: %v", allErrors)
	}

	return nil
}

func (q *RegisteredJobs) FindJobProvider(name string) JobProvider {
	provider, ok := q.providers[name]
	if !ok {
		return nil
	}

	return provider
}

func (q *RegisteredJobs) RunJobLoop(jctx JobRunContext, logger *slog.Logger, ctx context.Context) {

}

func (q *RegisteredJobs) fetchAvailableJobs(jctx JobRunContext, logger *slog.Logger, ctx context.Context, consumers chan availableJob) {
	for {
		time.Sleep(time.Second * 1)
		select {
		case <-ctx.Done():
			var ids []int64
			for job := range consumers {
				ids = append(ids, job.id)
			}
			if len(ids) == 0 {
				return
			}
			db := jctx.GetDB()
			query, args, err := sqlx.In("UPDATE jobs SET state = 'pending' WHERE id IN (?)", ids)
			if err != nil {
				logger.Error("Error preparing update query to revert jobs to pending", "err", err, "jobs", ids)
				return
			}
			query = db.Rebind(query)
			_, err = db.ExecContext(ctx, query, args...)
			if err != nil {
				logger.Error("Error updating job states to pending", "err", err, "jobs", ids)
			} else {
				logger.Info("Reverted jobs from queued to pending", "jobs", ids)
			}
			return
		default:
		}
		rows, err := jctx.GetDB().QueryxContext(ctx, `
		WITH selected_jobs AS (
			SELECT id
			FROM jobs
			WHERE state = 'pending'
			ORDER BY available_at ASC
		)
		UPDATE jobs
		SET state = 'queued'
		FROM selected_jobs
		WHERE jobs.id = selected_jobs.id
		RETURNING *;
		`)
		if err != nil {
			logger.Error("Error fetching available jobs", "err", err)
			continue
		}

		for rows.Next() {
			var job availableJob
			if err = rows.StructScan(&job); err != nil {
				logger.Error("Error scanning row", "err", err)
				continue
			}
			logger.Info("Queued job", "job-id", job.id)
			consumers <- job
		}

		rows.Close()
	}
}

type availableJob struct {
	id          int64     `db:"id"`
	name        string    `db:"name"`
	data        string    `db:"data"`
	state       JobState  `db:"job_state"`
	createdAt   time.Time `db:"created_at"`
	availableAt time.Time `db:"available_at"`
}
