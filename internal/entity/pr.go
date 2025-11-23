package entity

import "time"

// PRStatus - строгий тип для статуса PR
type PRStatus string

const (
	// StatusOpen - PR открыт и ждёт ревью
	StatusOpen PRStatus = "OPEN"
	// StatusMerged - PR влит изменения запрещены
	StatusMerged PRStatus = "MERGED"
)

// PullRequest - основная сущность задачи
type PullRequest struct {
	ID        string
	Name      string
	AuthorID  string
	Status    PRStatus
	Reviewers []string
	CreatedAt time.Time
	MergedAt  *time.Time
}

// CanBeMerged - мержить можно только открытый PR
func (pr *PullRequest) CanBeMerged() bool {
	return pr.Status == StatusOpen
}

// CanReassignReviewers - менять ревьюверов можно только пока PR открыт
func (pr *PullRequest) CanReassignReviewers() bool {
	return pr.Status == StatusOpen
}
