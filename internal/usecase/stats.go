package usecase

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReviewerStat хранит данные по количеству назначений
type ReviewerStat struct {
	UserID      string
	Username    string
	Assignments int64
}

// StatsService выдаёт статистику по ревьюверам
type StatsService interface {
	GetReviewerStats(ctx context.Context) ([]ReviewerStat, error)
}

type statsServiceImpl struct {
	db *pgxpool.Pool
}

// NewStatsService создаёт реализацию StatsService
func NewStatsService(db *pgxpool.Pool) StatsService {
	return &statsServiceImpl{db: db}
}

func (s *statsServiceImpl) GetReviewerStats(ctx context.Context) ([]ReviewerStat, error) {
	const query = `
SELECT u.id, u.username, COUNT(prr.pull_request_id) AS assignments
FROM users u
LEFT JOIN pr_reviewers prr ON prr.reviewer_id = u.id
GROUP BY u.id, u.username
ORDER BY assignments DESC, u.id
`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ReviewerStat
	for rows.Next() {
		var st ReviewerStat
		if err := rows.Scan(&st.UserID, &st.Username, &st.Assignments); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}
