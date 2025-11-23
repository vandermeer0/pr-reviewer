package usecase

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/vandermeer0/pr-reviewer/internal/entity"
	"github.com/vandermeer0/pr-reviewer/internal/usecase/repo"
)

type teamService struct {
	userRepo repo.UserRepository
	teamRepo repo.TeamRepository
}

// NewTeamService создаёт реализацию TeamService
func NewTeamService(userRepo repo.UserRepository, teamRepo repo.TeamRepository) TeamService {
	return &teamService{
		userRepo: userRepo,
		teamRepo: teamRepo,
	}
}

func (s *teamService) CreateTeam(
	ctx context.Context,
	teamName string,
	members []CreateTeamMemberInput,
) (*entity.Team, error) {
	team := &entity.Team{
		Name:    teamName,
		Members: make([]*entity.User, 0, len(members)),
	}

	users := make([]*entity.User, 0, len(members))
	for _, m := range members {
		u := &entity.User{
			ID:       m.UserID,
			Username: m.Username,
			TeamName: teamName,
			IsActive: m.IsActive,
		}
		if err := u.Validate(); err != nil {
			return nil, err
		}
		users = append(users, u)
		team.Members = append(team.Members, u)
	}

	if err := s.teamRepo.Save(ctx, team); err != nil {
		if errors.Is(err, repo.ErrAlreadyExists) {
			return nil, NewTeamExistsError("team already exists")
		}
		return nil, err
	}

	if err := s.userRepo.SaveBatch(ctx, users); err != nil {
		return nil, err
	}

	return team, nil
}

func (s *teamService) GetTeam(ctx context.Context, teamName string) (*entity.Team, error) {
	team, err := s.teamRepo.GetByName(ctx, teamName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewNotFoundError("team not found")
		}
		return nil, err
	}
	return team, nil
}

type userService struct {
	userRepo repo.UserRepository
}

// NewUserService создаёт реализацию UserService
func NewUserService(userRepo repo.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) SetIsActive(
	ctx context.Context,
	userID string,
	isActive bool,
) (*entity.User, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewNotFoundError("user not found")
		}
		return nil, err
	}

	u.IsActive = isActive
	if err := s.userRepo.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

type pullRequestService struct {
	prRepo   repo.PullRequestRepository
	userRepo repo.UserRepository
	teamRepo repo.TeamRepository
	rng      *rand.Rand
}

// NewPullRequestService создаёт реализацию PullRequestService
func NewPullRequestService(
	prRepo repo.PullRequestRepository,
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
) PullRequestService {
	return &pullRequestService{
		prRepo:   prRepo,
		userRepo: userRepo,
		teamRepo: teamRepo,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *pullRequestService) Create(
	ctx context.Context,
	input PullRequestCreateInput,
) (*entity.PullRequest, error) {
	author, err := s.userRepo.GetByID(ctx, input.AuthorID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewNotFoundError("author not found")
		}
		return nil, err
	}

	team, err := s.teamRepo.GetByName(ctx, author.TeamName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewNotFoundError("author team not found")
		}
		return nil, err
	}

	candidates := make([]string, 0, len(team.Members))
	for _, m := range team.Members {
		if m == nil {
			continue
		}
		if !m.IsActive {
			continue
		}
		if m.ID == author.ID {
			continue
		}
		candidates = append(candidates, m.ID)
	}

	if len(candidates) > 1 {
		s.rng.Shuffle(len(candidates), func(i, j int) {
			candidates[i], candidates[j] = candidates[j], candidates[i]
		})
	}

	reviewers := candidates
	if len(reviewers) > 2 {
		reviewers = reviewers[:2]
	}

	pr := &entity.PullRequest{
		ID:        input.ID,
		Name:      input.Name,
		AuthorID:  input.AuthorID,
		Status:    entity.StatusOpen,
		Reviewers: reviewers,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.prRepo.Save(ctx, pr); err != nil {
		if errors.Is(err, repo.ErrAlreadyExists) {
			return nil, NewPRExistsError("pull request already exists")
		}
		return nil, err
	}

	return pr, nil
}

func (s *pullRequestService) Merge(
	ctx context.Context,
	prID string,
) (*entity.PullRequest, error) {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewNotFoundError("pull request not found")
		}
		return nil, err
	}

	if !pr.CanBeMerged() {
		return pr, nil
	}

	now := time.Now().UTC()
	pr.Status = entity.StatusMerged
	pr.MergedAt = &now

	if err := s.prRepo.Update(ctx, pr); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewNotFoundError("pull request not found")
		}
		return nil, err
	}

	return pr, nil
}

func (s *pullRequestService) ReassignReviewer(
	ctx context.Context,
	prID string,
	oldReviewerID string,
) (*entity.PullRequest, string, error) {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, "", NewNotFoundError("pull request not found")
		}
		return nil, "", err
	}

	if !pr.CanReassignReviewers() {
		return nil, "", NewPRMergedError("pull request already merged")
	}

	index := -1
	for i, id := range pr.Reviewers {
		if id == oldReviewerID {
			index = i
			break
		}
	}
	if index == -1 {
		return nil, "", NewNotAssignedError("reviewer is not assigned to this pull request")
	}

	reviewer, err := s.userRepo.GetByID(ctx, oldReviewerID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, "", NewNotFoundError("reviewer not found")
		}
		return nil, "", err
	}

	team, err := s.teamRepo.GetByName(ctx, reviewer.TeamName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, "", NewNotFoundError("team not found")
		}
		return nil, "", err
	}

	current := make(map[string]struct{}, len(pr.Reviewers))
	for _, id := range pr.Reviewers {
		current[id] = struct{}{}
	}

	candidates := make([]*entity.User, 0, len(team.Members))
	for _, m := range team.Members {
		if m == nil {
			continue
		}
		if !m.IsActive {
			continue
		}
		if m.ID == reviewer.ID {
			continue
		}
		if m.ID == pr.AuthorID {
			continue
		}
		if _, exists := current[m.ID]; exists {
			continue
		}
		candidates = append(candidates, m)
	}

	if len(candidates) == 0 {
		return nil, "", NewNoCandidateError("no active replacement candidate in team")
	}

	candidate := candidates[s.rng.Intn(len(candidates))]

	pr.Reviewers[index] = candidate.ID

	if err := s.prRepo.Update(ctx, pr); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, "", NewNotFoundError("pull request not found")
		}
		return nil, "", err
	}

	return pr, candidate.ID, nil
}

func (s *pullRequestService) GetByReviewer(
	ctx context.Context,
	reviewerID string,
) ([]*entity.PullRequest, error) {
	prs, err := s.prRepo.GetByReviewerID(ctx, reviewerID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return []*entity.PullRequest{}, nil
		}
		return nil, err
	}
	return prs, nil
}
