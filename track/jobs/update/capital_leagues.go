package update

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MrNemo64/coc-tracker/track/jobs"
	"github.com/MrNemo64/coc-tracker/util"
	"github.com/jmoiron/sqlx"
)

type FetchCapitalLeagues struct{}
type FetchCapitalLeaguesProvider struct{}

func insertUpdateCapitalLeaguesJob(tx *sqlx.Tx, at *time.Time) error {
	defer tx.Rollback()

	_, err := tx.Exec("DELETE FROM jobs WHERE name = $1", "update/FetchCapitalLeagues")
	if err != nil {
		return err
	}

	if at == nil {
		_, err = tx.Exec("INSERT INTO jobs (name) VALUES ($1)", "update/FetchCapitalLeagues")
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec("INSERT INTO jobs (name, available_at) VALUES ($1, $2)", "update/FetchCapitalLeagues", at)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func NewFetchCapitalLeaguesProvider() *FetchCapitalLeaguesProvider {
	return &FetchCapitalLeaguesProvider{}
}

func (*FetchCapitalLeagues) Run(jctx jobs.JobRunContext, c context.Context) (*jobs.JobFinishInformation, error) {
	response, cacheHit, err := jctx.Get(c, util.CapitalLeagueEndpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return &jobs.JobFinishInformation{
			Successfull: false,
			Reschedule: &jobs.ScheduleInformation{
				At: time.Now().Add(time.Hour * 1),
			},
		}, nil
	}

	if cacheHit {
		return &jobs.JobFinishInformation{
			Successfull: true,
			Reschedule: &jobs.ScheduleInformation{
				At: time.Now().Add(time.Hour * 24 * 7),
			},
		}, nil
	}

	type TRes struct {
		Leagues []struct {
			Id   int    `json:"id" db:"id"`
			Name string `json:"name" db:"name"`
		} `json:"items"`
	}
	var leagues TRes

	if err = json.NewDecoder(response.Body).Decode(&leagues); err != nil {
		return nil, err
	}

	tx, err := jctx.GetDB().Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	stmt, err := tx.PrepareNamed(`
	INSERT INTO capital_leagues (id, name)
	VALUES (:id, :name)
	ON CONFLICT (id)
	DO UPDATE SET name = EXCLUDED.name;
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	for _, league := range leagues.Leagues {
		_, err := stmt.Exec(league)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &jobs.JobFinishInformation{
		Successfull: true,
		Reschedule: &jobs.ScheduleInformation{
			At: time.Now().Add(time.Hour * 24 * 7),
		},
	}, nil
}

func (*FetchCapitalLeagues) Serialize(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	return insertUpdateCapitalLeaguesJob(tx, nil)
}

func (*FetchCapitalLeaguesProvider) Deserialize(_ string) (jobs.Job, error) {
	return &FetchCapitalLeagues{}, nil
}

func (*FetchCapitalLeaguesProvider) Save(db *sqlx.Tx, info *jobs.ScheduleInformation) error {
	return insertUpdateCapitalLeaguesJob(db, &info.At)
}

func (j *FetchCapitalLeaguesProvider) CheckJobsTable(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	return insertUpdateCapitalLeaguesJob(tx, nil)
}

func (*FetchCapitalLeaguesProvider) JobName() string {
	return "update/FetchCapitalLeagues"
}
