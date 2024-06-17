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
	JobStatePending JobState = "pending"
	JobStateQueued  JobState = "queued"
	JobStateRunning JobState = "running"
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

func (q *RegisteredJobs) FetchAvailableJobs(jctx JobRunContext, logger *slog.Logger, ctx context.Context, consumers chan DBJob) error {
	for {
		time.Sleep(time.Second * 1)
		select {
		case <-ctx.Done():
			ids := collectJobsFromChannel(consumers)
			if len(ids) == 0 {
				return nil
			}
			return setJobsToPending(ids, logger, jctx.GetDB())
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

		var jobs []DBJob

		for rows.Next() {
			var job DBJob
			if err = rows.StructScan(&job); err != nil {
				logger.Error("Error scanning row", "err", err)
				continue
			}
			jobs = append(jobs, job)
		}

		if err := rows.Close(); err != nil {
			logger.Error("Error closing rows", "err", err)
		}

		for i := 0; i < len(jobs); i++ {
			job := jobs[i]
			select {
			case consumers <- job:
				logger.Info("Queued job", "job-id", job.Id)
			case <-ctx.Done():
				ids := collectJobsFromChannel(consumers)
				for ; i < len(jobs); i++ {
					ids = append(ids, jobs[i].Id)
				}
				if len(ids) == 0 {
					return nil
				}
				return setJobsToPending(ids, logger, jctx.GetDB())
			}
		}
	}
}

func collectJobsFromChannel(channel chan DBJob) []int64 {
	var ids []int64
	for {
		select {
		case job := <-channel:
			ids = append(ids, job.Id)
		default:
			return ids
		}
	}
}

func setJobsToPending(ids []int64, logger *slog.Logger, db *sqlx.DB) error {
	query, args, err := sqlx.In("UPDATE jobs SET state = 'pending' WHERE id IN (?)", ids)
	if err != nil {
		logger.Error("Error preparing update query to revert jobs to pending", "err", err, "jobs", ids)
		return err
	}
	query = db.Rebind(query)
	_, err = db.Exec(query, args...)
	if err != nil {
		logger.Error("Error updating job states to pending", "err", err, "jobs", ids)
		return err
	}
	logger.Info("Reverted jobs from queued to pending", "jobs", ids)
	return nil
}

type DBJob struct {
	Id          int64     `db:"id"`
	Name        string    `db:"name"`
	Data        string    `db:"data"`
	State       JobState  `db:"state"`
	CreatedAt   time.Time `db:"created_at"`
	AvailableAt time.Time `db:"available_at"`
}
