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

	return pool
}

func TestTeamMaintenance_Deactivate_NoPRs(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)

	_, err := pool.Exec(ctx, `
		INSERT INTO teams (name) VALUES ('tm_team1');
		INSERT INTO users (id, username, team_name, is_active) VALUES
			('tm_u1', 'Alice', 'tm_team1', TRUE),
			('tm_u2', 'Bob',   'tm_team1', TRUE);
	`)
	require.NoError(t, err)

	svc := NewTeamMaintenanceService(pool)
	res, err := svc.DeactivateTeamMembers(ctx, "tm_team1")
	require.NoError(t, err)

	require.Equal(t, "tm_team1", res.TeamName)

	var activeCount int64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE team_name = 'tm_team1' AND is_active = TRUE;
	`).Scan(&activeCount)
	require.NoError(t, err)
	require.EqualValues(t, 0, activeCount)
}

func TestTeamMaintenance_Deactivate_WithOpenPRsAndReassign(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)

	_, err := pool.Exec(ctx, `
		INSERT INTO teams (name) VALUES ('tm_authors'), ('tm_reviewers');

		INSERT INTO users (id, username, team_name, is_active) VALUES
			('tm_a1', 'Author1', 'tm_authors', TRUE),
			('tm_a2', 'Author2', 'tm_authors', TRUE),
			('tm_r1', 'Rev1',    'tm_reviewers', TRUE),
			('tm_r2', 'Rev2',    'tm_reviewers', TRUE);

		INSERT INTO pull_requests (id, name, author_id, status, created_at, merged_at) VALUES
			('tm_pr_open_1', 'Open 1', 'tm_a1', 'OPEN',   NOW(), NULL),
			('tm_pr_open_2', 'Open 2', 'tm_a2', 'OPEN',   NOW(), NULL),
			('tm_pr_merged', 'Merged', 'tm_a1', 'MERGED', NOW(), NOW());

		INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES
			('tm_pr_open_1',  'tm_r1'),
			('tm_pr_open_1',  'tm_r2'),
			('tm_pr_open_2',  'tm_r1'),
			('tm_pr_merged',  'tm_r1');
	`)
	require.NoError(t, err)

	svc := NewTeamMaintenanceService(pool)
	res, err := svc.DeactivateTeamMembers(ctx, "tm_reviewers")
	require.NoError(t, err)

	require.Equal(t, "tm_reviewers", res.TeamName)

	var activeReviewersTeamCount int64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE team_name = 'tm_reviewers' AND is_active = TRUE;
	`).Scan(&activeReviewersTeamCount)
	require.NoError(t, err)
	require.EqualValues(t, 0, activeReviewersTeamCount)

	var badAssignments int64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pr_reviewers r
		JOIN users u ON u.id = r.reviewer_id
		JOIN pull_requests p ON p.id = r.pull_request_id
		WHERE u.team_name = 'tm_reviewers'
		    AND p.status = 'OPEN';
	`).Scan(&badAssignments)
	require.NoError(t, err)
	require.EqualValues(t, 0, badAssignments)
}
