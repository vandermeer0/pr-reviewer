package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/vandermeer0/pr-reviewer/internal/config"
	"github.com/vandermeer0/pr-reviewer/internal/infrastructure/repository/postgresql"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	cfg := config.Load()
	pool, err := postgresql.NewPool(ctx, cfg.DB.ConnString())
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
	})

	_, err = pool.Exec(ctx, `
		TRUNCATE TABLE pr_reviewers, pull_requests, users, teams
		RESTART IDENTITY CASCADE;
	`)
	require.NoError(t, err)

	return pool
}

func TestTeamMaintenance_Deactivate_NoPRs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := newTestPool(t)

	_, err := pool.Exec(ctx, `
		INSERT INTO teams (name) VALUES ('team1');
		INSERT INTO users (id, username, team_name, is_active) VALUES
			('u1', 'Alice', 'team1', TRUE),
			('u2', 'Bob',   'team1', TRUE);
	`)
	require.NoError(t, err)

	svc := NewTeamMaintenanceService(pool)
	res, err := svc.DeactivateTeamMembers(ctx, "team1")
	require.NoError(t, err)

	require.Equal(t, "team1", res.TeamName)
	require.EqualValues(t, 2, res.DeactivatedUsers)
	require.EqualValues(t, 0, res.RemovedAssignments)
	require.EqualValues(t, 0, res.NewAssignments)
	require.Equal(t, 0, res.AffectedPullRequests)

	var activeCount int64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE team_name = 'team1' AND is_active = TRUE;
	`).Scan(&activeCount)
	require.NoError(t, err)
	require.EqualValues(t, 0, activeCount)
}

func TestTeamMaintenance_Deactivate_WithOpenPRsAndReassign(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := newTestPool(t)

	_, err := pool.Exec(ctx, `
		INSERT INTO teams (name) VALUES ('authors'), ('reviewers');

		INSERT INTO users (id, username, team_name, is_active) VALUES
			('a1', 'Author1', 'authors', TRUE),
			('a2', 'Author2', 'authors', TRUE),
			('r1', 'Rev1',    'reviewers', TRUE),
			('r2', 'Rev2',    'reviewers', TRUE);

		INSERT INTO pull_requests (id, name, author_id, status, created_at, merged_at) VALUES
			('pr-open-1', 'Open 1', 'a1', 'OPEN',   NOW(), NULL),
			('pr-open-2', 'Open 2', 'a2', 'OPEN',   NOW(), NULL),
			('pr-merged', 'Merged', 'a1', 'MERGED', NOW(), NOW());

		INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES
			('pr-open-1',  'r1'),
			('pr-open-1',  'r2'),
			('pr-open-2',  'r1'),
			('pr-merged',  'r1');
	`)
	require.NoError(t, err)

	svc := NewTeamMaintenanceService(pool)
	res, err := svc.DeactivateTeamMembers(ctx, "reviewers")
	require.NoError(t, err)

	require.Equal(t, "reviewers", res.TeamName)
	require.EqualValues(t, 2, res.DeactivatedUsers)
	require.EqualValues(t, 3, res.RemovedAssignments)
	require.EqualValues(t, 2, res.NewAssignments)
	require.EqualValues(t, 2, res.AffectedPullRequests)

	var activeReviewersTeamCount int64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE team_name = 'reviewers' AND is_active = TRUE;
	`).Scan(&activeReviewersTeamCount)
	require.NoError(t, err)
	require.EqualValues(t, 0, activeReviewersTeamCount)

	var badAssignments int64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pr_reviewers r
		JOIN users u ON u.id = r.reviewer_id
		JOIN pull_requests p ON p.id = r.pull_request_id
		WHERE u.team_name = 'reviewers'
		AND p.status = 'OPEN';
	`).Scan(&badAssignments)
	require.NoError(t, err)
	require.EqualValues(t, 0, badAssignments)
}
