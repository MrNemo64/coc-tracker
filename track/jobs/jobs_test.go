package jobs_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	testutil "github.com/MrNemo64/coc-tracker/test_util"
	"github.com/MrNemo64/coc-tracker/track/jobs"
	"github.com/MrNemo64/coc-tracker/util"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	util.LoadEnv()
	os.Exit(m.Run())
}

type mockJobRunContext struct {
	db *sqlx.DB
}

func (m *mockJobRunContext) GetDB() *sqlx.DB { return m.db }
func (m *mockJobRunContext) Get(context.Context, string) (*http.Response, bool, error) {
	return nil, false, fmt.Errorf("not implemented")
}

func TestFetchAvailableJobs(t *testing.T) {

	var prepareForTestCase = func(channelSize int) (container *testutil.TestDatabase, providers *jobs.RegisteredJobs, logger *testutil.TestLogBuffer, jobChannel chan jobs.DBJob, jctx *mockJobRunContext, ctx context.Context, cancel context.CancelFunc, mockjobs []jobs.DBJob) {

		container, err := testutil.CreatePostgresContainer()
		if err != nil {
			t.Errorf("Could not set up test database: %v", err)
		}

		mockjobs = []jobs.DBJob{
			{
				Id:          1,
				Name:        "Job 1",
				Data:        "{}",
				State:       jobs.JobStatePending,
				CreatedAt:   time.Now().Add(-5 * time.Hour),
				AvailableAt: time.Now().Add(-5 * time.Hour),
			},
			{
				Id:          2,
				Name:        "Job 2",
				Data:        "{}",
				State:       jobs.JobStatePending,
				CreatedAt:   time.Now().Add(-1 * time.Hour),
				AvailableAt: time.Now().Add(-1 * time.Hour),
			},
			{
				Id:          3,
				Name:        "Job 3",
				Data:        "{}",
				State:       jobs.JobStatePending,
				CreatedAt:   time.Now().Add(-3 * time.Hour),
				AvailableAt: time.Now().Add(-3 * time.Hour),
			},
			{
				Id:          4,
				Name:        "Job 4",
				Data:        "{}",
				State:       jobs.JobStatePending,
				CreatedAt:   time.Now().Add(-2 * time.Hour),
				AvailableAt: time.Now().Add(-2 * time.Hour),
			},
			{
				Id:          5,
				Name:        "Job 5",
				Data:        "{}",
				State:       jobs.JobStatePending,
				CreatedAt:   time.Now().Add(1 * time.Hour),
				AvailableAt: time.Now().Add(1 * time.Hour),
			},
		}

		// ORDER: Job 1 -> Job 3 -> Job 4 -> Job 2 -> Job 5

		for _, job := range mockjobs {
			if _, err := container.DB.NamedExec(
				`INSERT INTO jobs(id, name, data, state, created_at, available_at)
				VALUES (:id, :name, :data, :state, :created_at, :available_at)`,
				job); err != nil {
				t.Errorf("Error inserting job %v: %v", job, err)
			}
		}

		providers = jobs.NewJobQueue()
		logger = testutil.MakeTestLogger()
		jobChannel = make(chan jobs.DBJob, channelSize)
		jctx = &mockJobRunContext{
			db: container.DB,
		}
		ctx, cancel = context.WithCancel(context.Background())
		return
	}

	t.Run("Channel is emptied when cancelled while blocked after taking out one job", func(t *testing.T) {
		container, providers, logger, jobChannel, jctx, ctx, cancel, mockJobs := prepareForTestCase(2)
		t.Cleanup(func() {
			if err := container.Shutdown(); err != nil {
				t.Errorf("Error shuting down test container: %v", err)
			}
			cancel()
			close(jobChannel)
		})

		errChanel := make(chan error, 1)
		go func() {
			errChanel <- providers.FetchAvailableJobs(jctx, logger.Logger, ctx, jobChannel)
		}()

		select {
		case job := <-jobChannel:
			if _, err := container.DB.Exec(`UPDATE jobs SET state = $1 WHERE id = $2`, jobs.JobStateRunning, job.Id); err != nil {
				t.Errorf("Could not update state of extracted job: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("Timed out waiting for Job 1 to appear in channel")
		}

		cancel()
		select {
		case resultErr := <-errChanel:
			if resultErr != nil {
				t.Errorf("FetchAvailableJobs failed to cancel: %v", resultErr)
			}
		case <-time.After(3 * time.Second):
			t.Error("Timed out waiting for FetchAvailableJobs to cancel")
		}

		var foundJobs []jobs.DBJob
	channelCollectFor:
		for {
			select {
			case found := <-jobChannel:
				foundJobs = append(foundJobs, found)
			default:
				break channelCollectFor
			}
		}

		if len(foundJobs) > 0 {
			t.Errorf("Jobs channel is not empty after cancelation, found %v", foundJobs)
		}

		mockJobs[0].State = jobs.JobStateRunning
		container.AssertJobsTableEquals(t, mockJobs)
	})

	t.Run("Consume all four available jobs in order", func(t *testing.T) {
		container, providers, logger, jobChannel, jctx, ctx, cancel, mockJobs := prepareForTestCase(2)
		t.Cleanup(func() {
			if err := container.Shutdown(); err != nil {
				t.Errorf("Error shuting down test container: %v", err)
			}
			cancel()
			close(jobChannel)
		})

		errChanel := make(chan error, 1)
		go func() {
			errChanel <- providers.FetchAvailableJobs(jctx, logger.Logger, ctx, jobChannel)
		}()

		expectedOrder := []int64{1, 3, 4, 2}
		for i := 0; i < 4; i++ {
			select {
			case job := <-jobChannel:
				if _, err := container.DB.Exec(`UPDATE jobs SET state = $1 WHERE id = $2`, jobs.JobStateRunning, job.Id); err != nil {
					t.Errorf("Could not update state of extracted job: %v", err)
				}
				assert.Equal(t, expectedOrder[i], job.Id, "Did not get expected job")
			case <-time.After(3 * time.Second):
				t.Errorf("Timed out waiting for Job %d to appear in channel", i)
			}
		}

		select {
		case job := <-jobChannel:
			t.Errorf("Found job %v in channel when it should have been empty", job)
		default:
		}

		cancel()
		select {
		case resultErr := <-errChanel:
			if resultErr != nil {
				t.Errorf("FetchAvailableJobs failed to cancel: %v", resultErr)
			}
		case <-time.After(3 * time.Second):
			t.Error("Timed out waiting for FetchAvailableJobs to cancel")
		}

		var foundJobs []jobs.DBJob
	channelCollectFor:
		for {
			select {
			case found := <-jobChannel:
				foundJobs = append(foundJobs, found)
			default:
				break channelCollectFor
			}
		}

		if len(foundJobs) > 0 {
			t.Errorf("Jobs channel is not empty after cancelation, found %v", foundJobs)
		}

		mockJobs[0].State = jobs.JobStateRunning
		mockJobs[2].State = jobs.JobStateRunning
		mockJobs[3].State = jobs.JobStateRunning
		mockJobs[1].State = jobs.JobStateRunning
		container.AssertJobsTableEquals(t, mockJobs)
	})
}
