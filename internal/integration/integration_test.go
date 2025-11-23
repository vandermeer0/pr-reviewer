package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vandermeer0/pr-reviewer/internal/config"
	"github.com/vandermeer0/pr-reviewer/internal/infrastructure/repository/postgresql"
	"github.com/vandermeer0/pr-reviewer/internal/transport/httpapi"
	"github.com/vandermeer0/pr-reviewer/internal/usecase"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	cfg := config.Load()

	pool, err := postgresql.NewPool(ctx, cfg.DB.ConnString())
	require.NoError(t, err, "failed to create pgx pool")
	t.Cleanup(func() {
		pool.Close()
	})

	userRepo := postgresql.NewUserRepository(pool)
	teamRepo := postgresql.NewTeamRepository(pool)
	prRepo := postgresql.NewPullRequestRepository(pool)

	teamSvc := usecase.NewTeamService(userRepo, teamRepo)
	userSvc := usecase.NewUserService(userRepo)
	prSvc := usecase.NewPullRequestService(prRepo, userRepo, teamRepo)
	statsSvc := usecase.NewStatsService(pool)
	teamMaintSvc := usecase.NewTeamMaintenanceService(pool)

	apiServer := httpapi.NewServer(teamSvc, userSvc, prSvc, statsSvc, teamMaintSvc)
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv
}

func TestIntegration_FullFlow(t *testing.T) {
	srv := newTestServer(t)
	client := srv.Client()
	baseURL := srv.URL

	resp, err := client.Get(baseURL + "/health")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, resp.Body.Close())

	teamBody := []byte(`{
                "team_name": "backend",
                "members": [
                        { "user_id": "u1", "username": "Alice", "is_active": true },
                        { "user_id": "u2", "username": "Bob",   "is_active": true },
                        { "user_id": "u3", "username": "Carol", "is_active": true }
                ]
        }`)
	resp, err = client.Post(baseURL+"/team/add", "application/json", bytes.NewReader(teamBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NoError(t, resp.Body.Close())

	prBody := []byte(`{
                "pull_request_id": "pr-1001",
                "pull_request_name": "Add search",
                "author_id": "u1"
        }`)
	resp, err = client.Post(baseURL+"/pullRequest/create", "application/json", bytes.NewReader(prBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var prResp struct {
		PR struct {
			PullRequestID     string   `json:"pull_request_id"`
			PullRequestName   string   `json:"pull_request_name"`
			AuthorID          string   `json:"author_id"`
			Status            string   `json:"status"`
			AssignedReviewers []string `json:"assigned_reviewers"`
		} `json:"pr"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&prResp))
	require.NoError(t, resp.Body.Close())

	require.Equal(t, "pr-1001", prResp.PR.PullRequestID)
	require.Equal(t, "Add search", prResp.PR.PullRequestName)
	require.Equal(t, "u1", prResp.PR.AuthorID)
	require.Equal(t, "OPEN", prResp.PR.Status)

	require.True(t, len(prResp.PR.AssignedReviewers) >= 1 && len(prResp.PR.AssignedReviewers) <= 2)
	for _, r := range prResp.PR.AssignedReviewers {
		require.NotEqual(t, "u1", r)
	}

	resp, err = client.Get(baseURL + "/users/getReview?user_id=u1")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var reviewResp struct {
		UserID       string `json:"user_id"`
		PullRequests []struct {
			PullRequestID string `json:"pull_request_id"`
		} `json:"pull_requests"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reviewResp))
	require.NoError(t, resp.Body.Close())

	require.Equal(t, "u1", reviewResp.UserID)
	require.Len(t, reviewResp.PullRequests, 0)

	resp, err = client.Get(baseURL + "/stats/reviewers")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var statsResp struct {
		Reviewers []struct {
			UserID      string `json:"user_id"`
			Username    string `json:"username"`
			Assignments int64  `json:"assignments"`
		} `json:"reviewers"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&statsResp))
	require.NoError(t, resp.Body.Close())

	require.GreaterOrEqual(t, len(statsResp.Reviewers), 3)

	expected := map[string]bool{
		"u1": false,
		"u2": false,
		"u3": false,
	}
	for _, r := range statsResp.Reviewers {
		if _, ok := expected[r.UserID]; ok {
			expected[r.UserID] = true
		}
	}

	for id, found := range expected {
		require.Truef(t, found, "expected reviewer %s in stats", id)
	}
}
