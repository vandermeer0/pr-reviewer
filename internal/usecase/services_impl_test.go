package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vandermeer0/pr-reviewer/internal/entity"
	"github.com/vandermeer0/pr-reviewer/internal/usecase/repo"
)

type inMemoryUserRepo struct {
	users map[string]*entity.User
}

func newInMemoryUserRepo() *inMemoryUserRepo {
	return &inMemoryUserRepo{
		users: make(map[string]*entity.User),
	}
}

func (r *inMemoryUserRepo) Save(_ context.Context, u *entity.User) error {
	uCopy := *u
	r.users[u.ID] = &uCopy
	return nil
}

func (r *inMemoryUserRepo) GetByID(_ context.Context, id string) (*entity.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, repo.ErrNotFound
	}
	uCopy := *u
	return &uCopy, nil
}

func (r *inMemoryUserRepo) SaveBatch(ctx context.Context, users []*entity.User) error {
	for _, u := range users {
		if err := r.Save(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

type inMemoryTeamRepo struct {
	teams map[string]*entity.Team
}

func newInMemoryTeamRepo() *inMemoryTeamRepo {
	return &inMemoryTeamRepo{
		teams: make(map[string]*entity.Team),
	}
}

func (r *inMemoryTeamRepo) Save(_ context.Context, team *entity.Team) error {
	if _, exists := r.teams[team.Name]; exists {
		return repo.ErrAlreadyExists
	}
	teamCopy := *team
	r.teams[team.Name] = &teamCopy
	return nil
}

func (r *inMemoryTeamRepo) GetByName(_ context.Context, name string) (*entity.Team, error) {
	team, ok := r.teams[name]
	if !ok {
		return nil, repo.ErrNotFound
	}
	teamCopy := *team
	teamCopy.Members = make([]*entity.User, 0, len(team.Members))
	for _, m := range team.Members {
		if m == nil {
			continue
		}
		u := *m
		teamCopy.Members = append(teamCopy.Members, &u)
	}
	return &teamCopy, nil
}

type inMemoryPRRepo struct {
	prs map[string]*entity.PullRequest
}

func newInMemoryPRRepo() *inMemoryPRRepo {
	return &inMemoryPRRepo{
		prs: make(map[string]*entity.PullRequest),
	}
}

func (r *inMemoryPRRepo) Save(_ context.Context, pr *entity.PullRequest) error {
	if pr == nil {
		return errors.New("pr is nil")
	}
	if _, exists := r.prs[pr.ID]; exists {
		return repo.ErrAlreadyExists
	}
	prCopy := *pr
	prCopy.Reviewers = append([]string(nil), pr.Reviewers...)
	r.prs[pr.ID] = &prCopy
	return nil
}

func (r *inMemoryPRRepo) GetByID(_ context.Context, id string) (*entity.PullRequest, error) {
	pr, ok := r.prs[id]
	if !ok {
		return nil, repo.ErrNotFound
	}
	prCopy := *pr
	prCopy.Reviewers = append([]string(nil), pr.Reviewers...)
	return &prCopy, nil
}

func (r *inMemoryPRRepo) Update(_ context.Context, pr *entity.PullRequest) error {
	if pr == nil {
		return errors.New("pr is nil")
	}
	if _, exists := r.prs[pr.ID]; !exists {
		return repo.ErrNotFound
	}
	prCopy := *pr
	prCopy.Reviewers = append([]string(nil), pr.Reviewers...)
	r.prs[pr.ID] = &prCopy
	return nil
}

func (r *inMemoryPRRepo) GetByReviewerID(_ context.Context, reviewerID string) ([]*entity.PullRequest, error) {
	var result []*entity.PullRequest
	for _, pr := range r.prs {
		for _, rid := range pr.Reviewers {
			if rid == reviewerID {
				prCopy := *pr
				prCopy.Reviewers = append([]string(nil), pr.Reviewers...)
				result = append(result, &prCopy)
				break
			}
		}
	}
	if len(result) == 0 {
		return nil, repo.ErrNotFound
	}
	return result, nil
}

func TestPullRequestService_Create_AssignsReviewers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	makeService := func(t *testing.T, users []*entity.User, team *entity.Team) PullRequestService {
		t.Helper()
		ur := newInMemoryUserRepo()
		tr := newInMemoryTeamRepo()
		pr := newInMemoryPRRepo()

		for _, u := range users {
			require.NoError(t, ur.Save(ctx, u))
		}
		require.NoError(t, tr.Save(ctx, team))

		return NewPullRequestService(pr, ur, tr)
	}

	t.Run("no candidates (only author)", func(t *testing.T) {
		t.Parallel()
		author := &entity.User{ID: "u1", Username: "Alice", TeamName: "t", IsActive: true}
		team := &entity.Team{
			Name:    "t",
			Members: []*entity.User{author},
		}

		svc := makeService(t, []*entity.User{author}, team)

		pr, err := svc.Create(ctx, PullRequestCreateInput{
			ID:       "pr-no-candidates",
			Name:     "Test",
			AuthorID: "u1",
		})
		require.NoError(t, err)
		require.Equal(t, entity.StatusOpen, pr.Status)
		require.Len(t, pr.Reviewers, 0)
	})

	t.Run("one candidate", func(t *testing.T) {
		t.Parallel()
		author := &entity.User{ID: "u1", Username: "Alice", TeamName: "t", IsActive: true}
		u2 := &entity.User{ID: "u2", Username: "Bob", TeamName: "t", IsActive: true}
		team := &entity.Team{
			Name:    "t",
			Members: []*entity.User{author, u2},
		}

		svc := makeService(t, []*entity.User{author, u2}, team)

		pr, err := svc.Create(ctx, PullRequestCreateInput{
			ID:       "pr-one-candidate",
			Name:     "Test",
			AuthorID: "u1",
		})
		require.NoError(t, err)
		require.Len(t, pr.Reviewers, 1)
		require.Equal(t, "u2", pr.Reviewers[0])
	})

	t.Run("two active candidates, inactive и автор отфильтрованы", func(t *testing.T) {
		t.Parallel()
		author := &entity.User{ID: "u1", Username: "Alice", TeamName: "t", IsActive: true}
		u2 := &entity.User{ID: "u2", Username: "Bob", TeamName: "t", IsActive: true}
		u3 := &entity.User{ID: "u3", Username: "Carol", TeamName: "t", IsActive: false}
		u4 := &entity.User{ID: "u4", Username: "Dave", TeamName: "t", IsActive: true}

		team := &entity.Team{
			Name:    "t",
			Members: []*entity.User{author, u2, u3, u4},
		}

		svc := makeService(t, []*entity.User{author, u2, u3, u4}, team)

		pr, err := svc.Create(ctx, PullRequestCreateInput{
			ID:       "pr-two-candidates",
			Name:     "Test",
			AuthorID: "u1",
		})
		require.NoError(t, err)
		require.Len(t, pr.Reviewers, 2)

		for _, rID := range pr.Reviewers {
			require.NotEqual(t, "u1", rID)
			require.NotEqual(t, "u3", rID)
		}
	})
}

func TestPullRequestService_ReassignReviewer_HappyPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ur := newInMemoryUserRepo()
	tr := newInMemoryTeamRepo()
	prr := newInMemoryPRRepo()

	author := &entity.User{ID: "u-author", Username: "Author", TeamName: "team", IsActive: true}
	oldRev := &entity.User{ID: "u-old", Username: "Old", TeamName: "team", IsActive: true}
	newRev := &entity.User{ID: "u-new", Username: "New", TeamName: "team", IsActive: true}

	for _, u := range []*entity.User{author, oldRev, newRev} {
		require.NoError(t, ur.Save(ctx, u))
	}

	team := &entity.Team{
		Name:    "team",
		Members: []*entity.User{author, oldRev, newRev},
	}
	require.NoError(t, tr.Save(ctx, team))

	pr := &entity.PullRequest{
		ID:        "pr-1",
		Name:      "Test PR",
		AuthorID:  author.ID,
		Status:    entity.StatusOpen,
		Reviewers: []string{oldRev.ID},
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, prr.Save(ctx, pr))

	svc := NewPullRequestService(prr, ur, tr)
	prOut, replacedBy, err := svc.ReassignReviewer(ctx, "pr-1", oldRev.ID)
	require.NoError(t, err)
	require.Equal(t, "pr-1", prOut.ID)
	require.Len(t, prOut.Reviewers, 1)

	require.Equal(t, replacedBy, prOut.Reviewers[0])
	require.Equal(t, newRev.ID, replacedBy)
	require.NotEqual(t, oldRev.ID, replacedBy)
	require.NotEqual(t, author.ID, replacedBy)
}

func TestPullRequestService_ReassignReviewer_Errors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("PR_MERGED", func(t *testing.T) {
		ur := newInMemoryUserRepo()
		tr := newInMemoryTeamRepo()
		prr := newInMemoryPRRepo()

		author := &entity.User{ID: "a", Username: "A", TeamName: "team", IsActive: true}
		rev := &entity.User{ID: "r", Username: "R", TeamName: "team", IsActive: true}

		for _, u := range []*entity.User{author, rev} {
			require.NoError(t, ur.Save(ctx, u))
		}
		require.NoError(t, tr.Save(ctx, &entity.Team{
			Name:    "team",
			Members: []*entity.User{author, rev},
		}))

		pr := &entity.PullRequest{
			ID:        "pr-merged",
			Name:      "Merged",
			AuthorID:  author.ID,
			Status:    entity.StatusMerged,
			Reviewers: []string{rev.ID},
			CreatedAt: time.Now().UTC(),
		}
		require.NoError(t, prr.Save(ctx, pr))

		svc := NewPullRequestService(prr, ur, tr)
		_, _, err := svc.ReassignReviewer(ctx, "pr-merged", rev.ID)
		require.Error(t, err)

		var de *DomainError
		require.ErrorAs(t, err, &de)
		require.Equal(t, ErrorCodePRMerged, de.Code)
	})

	t.Run("NOT_ASSIGNED", func(t *testing.T) {
		ur := newInMemoryUserRepo()
		tr := newInMemoryTeamRepo()
		prr := newInMemoryPRRepo()

		author := &entity.User{ID: "a", Username: "A", TeamName: "team", IsActive: true}
		revAssigned := &entity.User{ID: "r1", Username: "R1", TeamName: "team", IsActive: true}
		revNotAssigned := &entity.User{ID: "r2", Username: "R2", TeamName: "team", IsActive: true}

		for _, u := range []*entity.User{author, revAssigned, revNotAssigned} {
			require.NoError(t, ur.Save(ctx, u))
		}
		require.NoError(t, tr.Save(ctx, &entity.Team{
			Name:    "team",
			Members: []*entity.User{author, revAssigned, revNotAssigned},
		}))

		pr := &entity.PullRequest{
			ID:        "pr-na",
			Name:      "NA",
			AuthorID:  author.ID,
			Status:    entity.StatusOpen,
			Reviewers: []string{revAssigned.ID},
			CreatedAt: time.Now().UTC(),
		}
		require.NoError(t, prr.Save(ctx, pr))

		svc := NewPullRequestService(prr, ur, tr)
		_, _, err := svc.ReassignReviewer(ctx, "pr-na", revNotAssigned.ID)
		require.Error(t, err)

		var de *DomainError
		require.ErrorAs(t, err, &de)
		require.Equal(t, ErrorCodeNotAssigned, de.Code)
	})

	t.Run("NO_CANDIDATE", func(t *testing.T) {
		ur := newInMemoryUserRepo()
		tr := newInMemoryTeamRepo()
		prr := newInMemoryPRRepo()

		author := &entity.User{ID: "a", Username: "A", TeamName: "team", IsActive: true}
		rev1 := &entity.User{ID: "r1", Username: "R1", TeamName: "team", IsActive: true}

		for _, u := range []*entity.User{author, rev1} {
			require.NoError(t, ur.Save(ctx, u))
		}
		require.NoError(t, tr.Save(ctx, &entity.Team{
			Name:    "team",
			Members: []*entity.User{author, rev1},
		}))

		pr := &entity.PullRequest{
			ID:        "pr-nc",
			Name:      "NC",
			AuthorID:  author.ID,
			Status:    entity.StatusOpen,
			Reviewers: []string{rev1.ID},
			CreatedAt: time.Now().UTC(),
		}
		require.NoError(t, prr.Save(ctx, pr))

		svc := NewPullRequestService(prr, ur, tr)
		_, _, err := svc.ReassignReviewer(ctx, "pr-nc", rev1.ID)
		require.Error(t, err)

		var de *DomainError
		require.ErrorAs(t, err, &de)
		require.Equal(t, ErrorCodeNoCandidate, de.Code)
	})

	t.Run("NOT_FOUND (PR)", func(t *testing.T) {
		ur := newInMemoryUserRepo()
		tr := newInMemoryTeamRepo()
		prr := newInMemoryPRRepo()

		svc := NewPullRequestService(prr, ur, tr)
		_, _, err := svc.ReassignReviewer(ctx, "no-such-pr", "someone")
		require.Error(t, err)

		var de *DomainError
		require.ErrorAs(t, err, &de)
		require.Equal(t, ErrorCodeNotFound, de.Code)
	})
}

// --- TeamService.CreateTeam ---

func TestTeamService_CreateTeam_BasicAndErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		ur := newInMemoryUserRepo()
		tr := newInMemoryTeamRepo()

		svc := NewTeamService(ur, tr)

		team, err := svc.CreateTeam(ctx, "backend", []CreateTeamMemberInput{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: false},
		})
		require.NoError(t, err)
		require.Equal(t, "backend", team.Name)
		require.Len(t, team.Members, 2)

		require.Len(t, ur.users, 2)
		require.Contains(t, ur.users, "u1")
		require.Contains(t, ur.users, "u2")
	})

	t.Run("TEAM_EXISTS", func(t *testing.T) {
		ur := newInMemoryUserRepo()
		tr := newInMemoryTeamRepo()
		svc := NewTeamService(ur, tr)

		_, err := svc.CreateTeam(ctx, "backend", []CreateTeamMemberInput{
			{UserID: "u1", Username: "Alice", IsActive: true},
		})
		require.NoError(t, err)

		_, err = svc.CreateTeam(ctx, "backend", []CreateTeamMemberInput{
			{UserID: "u2", Username: "Bob", IsActive: true},
		})
		require.Error(t, err)

		var de *DomainError
		require.ErrorAs(t, err, &de)
		require.Equal(t, ErrorCodeTeamExists, de.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		ur := newInMemoryUserRepo()
		tr := newInMemoryTeamRepo()
		svc := NewTeamService(ur, tr)

		_, err := svc.CreateTeam(ctx, "backend", []CreateTeamMemberInput{
			{UserID: "", Username: "Alice", IsActive: true},
		})
		require.Error(t, err)

		var de *DomainError
		require.False(t, errors.As(err, &de), "validation не должна мапиться в DomainError")
	})
}
