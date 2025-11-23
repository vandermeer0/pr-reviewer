package postgresql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vandermeer0/pr-reviewer/internal/entity"
	"github.com/vandermeer0/pr-reviewer/internal/usecase/repo"
)

var (
	_ repo.UserRepository        = (*UserRepository)(nil)
	_ repo.TeamRepository        = (*TeamRepository)(nil)
	_ repo.PullRequestRepository = (*PullRequestRepository)(nil)
)

// UserRepository реализует repo.UserRepository с использованием PostgreSQL
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создает новый UserRepository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Save сохраняет или обновляет пользователя
func (r *UserRepository) Save(ctx context.Context, user *entity.User) error {
	if user == nil {
		return errors.New("user is nil")
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET username = EXCLUDED.username,
		    team_name = EXCLUDED.team_name,
		    is_active = EXCLUDED.is_active
	`, user.ID, user.Username, user.TeamName, user.IsActive)
	return err
}

// GetByID возвращает пользователя по идентификатору
func (r *UserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, username, team_name, is_active
		FROM users
		WHERE id = $1
	`, id)

	var u entity.User
	if err := row.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}

// SaveBatch сохраняет или обновляет список пользователей
func (r *UserRepository) SaveBatch(ctx context.Context, users []*entity.User) error {
	for _, u := range users {
		if err := r.Save(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

// TeamRepository реализует repo.TeamRepository с использованием PostgreSQL
type TeamRepository struct {
	pool *pgxpool.Pool
}

// NewTeamRepository создает новый TeamRepository
func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

// Save создает новую команду
func (r *TeamRepository) Save(ctx context.Context, team *entity.Team) error {
	if team == nil {
		return errors.New("team is nil")
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO teams (name)
		VALUES ($1)
	`, team.Name)
	if err != nil {
		if isUniqueViolation(err) {
			return repo.ErrAlreadyExists
		}
		return err
	}

	return nil
}

// GetByName возвращает команду с участниками
func (r *TeamRepository) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT name
		FROM teams
		WHERE name = $1
	`, name)

	var team entity.Team
	if err := row.Scan(&team.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, username, team_name, is_active
		FROM users
		WHERE team_name = $1
	`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u entity.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, err
		}
		team.Members = append(team.Members, &u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &team, nil
}

// PullRequestRepository реализует repo.PullRequestRepository с использованием PostgreSQL
type PullRequestRepository struct {
	pool *pgxpool.Pool
}

// NewPullRequestRepository создает новый PullRequestRepository
func NewPullRequestRepository(pool *pgxpool.Pool) *PullRequestRepository {
	return &PullRequestRepository{pool: pool}
}

// Save создает новый PR и его ревьюверов
func (r *PullRequestRepository) Save(ctx context.Context, pr *entity.PullRequest) error {
	if pr == nil {
		return errors.New("pull request is nil")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	_, err = tx.Exec(ctx, `
                INSERT INTO pull_requests (id, name, author_id, status, created_at, merged_at)
                VALUES ($1, $2, $3, $4, $5, $6)
        `, pr.ID, pr.Name, pr.AuthorID, string(pr.Status), pr.CreatedAt, pr.MergedAt)
	if err != nil {
		if isUniqueViolation(err) {
			_ = tx.Rollback(ctx)
			return repo.ErrAlreadyExists
		}
		_ = tx.Rollback(ctx)
		return err
	}

	if len(pr.Reviewers) > 0 {
		for _, reviewerID := range pr.Reviewers {
			_, err = tx.Exec(ctx, `
                        INSERT INTO pr_reviewers (pull_request_id, reviewer_id)
                        VALUES ($1, $2)
                `, pr.ID, reviewerID)
			if err != nil {
				_ = tx.Rollback(ctx)
				return err
			}
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

// GetByID возвращает PR с ревьюверами
func (r *PullRequestRepository) GetByID(ctx context.Context, id string) (*entity.PullRequest, error) {
	row := r.pool.QueryRow(ctx, `
                SELECT id, name, author_id, status, created_at, merged_at
                FROM pull_requests
                WHERE id = $1
        `, id)

	var pr entity.PullRequest
	var status string
	if err := row.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &status, &pr.CreatedAt, &pr.MergedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	pr.Status = entity.PRStatus(status)

	// Загружаем ID ревьюверов
	rows, err := r.pool.Query(ctx, `
                SELECT reviewer_id
                FROM pr_reviewers
                WHERE pull_request_id = $1
        `, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		pr.Reviewers = append(pr.Reviewers, reviewerID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &pr, nil
}

// Update обновляет данные PR и его ревьюверов
func (r *PullRequestRepository) Update(ctx context.Context, pr *entity.PullRequest) error {
	if pr == nil {
		return errors.New("pull request is nil")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	cmdTag, err := tx.Exec(ctx, `
                UPDATE pull_requests
                SET name = $2,
                    author_id = $3,
                    status = $4,
                    created_at = $5,
                    merged_at = $6
                WHERE id = $1
        `, pr.ID, pr.Name, pr.AuthorID, string(pr.Status), pr.CreatedAt, pr.MergedAt)
	if err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		_ = tx.Rollback(ctx)
		return repo.ErrNotFound
	}

	_, err = tx.Exec(ctx, `
                DELETE FROM pr_reviewers
                WHERE pull_request_id = $1
        `, pr.ID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if len(pr.Reviewers) > 0 {
		for _, reviewerID := range pr.Reviewers {
			_, err = tx.Exec(ctx, `
                        INSERT INTO pr_reviewers (pull_request_id, reviewer_id)
                        VALUES ($1, $2)
                `, pr.ID, reviewerID)
			if err != nil {
				_ = tx.Rollback(ctx)
				return err
			}
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

// GetByReviewerID возвращает PR, где указанный пользователь назначен ревьювером
func (r *PullRequestRepository) GetByReviewerID(ctx context.Context, reviewerID string) ([]*entity.PullRequest, error) {
	rows, err := r.pool.Query(ctx, `
                SELECT p.id, p.name, p.author_id, p.status, p.created_at, p.merged_at
                FROM pull_requests p
                JOIN pr_reviewers rvr ON p.id = rvr.pull_request_id
                WHERE rvr.reviewer_id = $1
                ORDER BY p.created_at DESC
        `, reviewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*entity.PullRequest

	for rows.Next() {
		var pr entity.PullRequest
		var status string
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &status, &pr.CreatedAt, &pr.MergedAt); err != nil {
			return nil, err
		}
		pr.Status = entity.PRStatus(status)
		result = append(result, &pr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
