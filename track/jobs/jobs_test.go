package jobs_test

import (
	"os"
	"testing"
	"time"

	testutil "github.com/MrNemo64/coc-tracker/test_util"
	"github.com/MrNemo64/coc-tracker/util"
)

func TestMain(m *testing.M) {
	util.LoadEnv()
	os.Exit(m.Run())
}

func TestFetchAvailableJobs(t *testing.T) {
	container, err := testutil.CreatePostgresContainer()
	if err != nil {
		t.Errorf("Could not set up test database: %v", err)
	}

	time.Sleep(5 * time.Second)

	t.Cleanup(func() {
		if err := container.Shutdown(); err != nil {
			t.Errorf("Error shuting down test container: %v", err)
		}
	})
}
