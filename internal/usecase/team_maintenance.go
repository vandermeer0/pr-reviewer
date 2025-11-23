package usecase

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TeamDeactivationResult описывает результат массовой деактивации участников
type TeamDeactivationResult struct {
	TeamName             string
	DeactivatedUsers     int64
	RemovedAssignments   int64
	NewAssignments       int64
	AffectedPullRequests int
}

// TeamMaintenanceService описывает операции по массовому обслуживанию команд
type TeamMaintenanceService interface {
	DeactivateTeamMembers(ctx context.Context, teamName string) (TeamDeactivationResult, error)
}

type teamMaintenanceServiceImpl struct {
	db *pgxpool.Pool
}

// NewTeamMaintenanceService создаёт реализацию TeamMaintenanceService
func NewTeamMaintenanceService(db *pgxpool.Pool) TeamMaintenanceService {
	return &teamMaintenanceServiceImpl{db: db}
}

func (s *teamMaintenanceServiceImpl) DeactivateTeamMembers(ctx context.Context, teamName string) (res TeamDeactivationResult, err error) {
	res.TeamName = teamName

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return res, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	cmd, err := tx.Exec(ctx, `
			UPDATE users
			SET is_active = FALSE
			WHERE team_name = $1
	`, teamName)
	if err != nil {
		return res, err
	}
	res.DeactivatedUsers = cmd.RowsAffected()

	rows, err := tx.Query(ctx, `
			DELETE FROM pr_reviewers prr
			USING pull_requests pr, users reviewer
			WHERE prr.pull_request_id = pr.id
				AND reviewer.id = prr.reviewer_id
				AND pr.status = 'OPEN'
				AND reviewer.team_name = $1
				AND reviewer.is_active = FALSE
			RETURNING prr.pull_request_id
	`, teamName)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	prSet := make(map[string]struct{})
	for rows.Next() {
		var prID string
		if err = rows.Scan(&prID); err != nil {
			return res, err
		}
		res.RemovedAssignments++
		prSet[prID] = struct{}{}
	}
	if err = rows.Err(); err != nil {
		return res, err
	}

	if len(prSet) == 0 {
		if err = tx.Commit(ctx); err != nil {
			return res, err
		}
		return res, nil
	}

	prIDs := make([]string, 0, len(prSet))
	for id := range prSet {
		prIDs = append(prIDs, id)
	}
	res.AffectedPullRequests = len(prIDs)

	cmd, err = tx.Exec(ctx, `
WITH affected AS (
    SELECT DISTINCT UNNEST($1::text[]) AS pr_id
),
current AS (
    SELECT
        pr.id AS pr_id,
        COUNT(prr.reviewer_id) AS existing_cnt
    FROM pull_requests pr
    LEFT JOIN pr_reviewers prr ON prr.pull_request_id = pr.id
    WHERE pr.id IN (SELECT pr_id FROM affected)
        AND pr.status = 'OPEN'
    GROUP BY pr.id
),
candidates AS (
    SELECT
        pr.id AS pr_id,
        u.id AS candidate_id,
        ROW_NUMBER() OVER (PARTITION BY pr.id ORDER BY RANDOM()) AS rn
    FROM pull_requests pr
    JOIN affected a ON a.pr_id = pr.id
    JOIN users author ON author.id = pr.author_id
    JOIN users u
        ON u.team_name = author.team_name
        AND u.is_active = TRUE
        AND u.id <> author.id
    WHERE NOT EXISTS (
        SELECT 1
        FROM pr_reviewers ex
        WHERE ex.pull_request_id = pr.id
            AND ex.reviewer_id = u.id
    )
),
to_insert AS (
    SELECT c.pr_id, c.candidate_id
    FROM candidates c
    JOIN current cur ON cur.pr_id = c.pr_id
    WHERE cur.existing_cnt < 2
        AND c.rn <= 2 - cur.existing_cnt
)
INSERT INTO pr_reviewers (pull_request_id, reviewer_id)
SELECT pr_id, candidate_id
FROM to_insert
`, prIDs)
	if err != nil {
		return res, err
	}
	res.NewAssignments = cmd.RowsAffected()

	if err = tx.Commit(ctx); err != nil {
		return res, err
	}

	return res, nil
}
