package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrUnauthorized = errors.New("github unauthorized")
	ErrNotFound     = errors.New("github not found")
	ErrForbidden    = errors.New("github forbidden")
	ErrRateLimited  = errors.New("github rate limited")
	ErrRetryable    = errors.New("github retryable error")
	ErrPermanent    = errors.New("github permanent error")
)

// Repository is the subset of GitHub repo fields we use.
type Repository struct {
	ID            int64       `json:"id"`
	Name          string      `json:"name"`
	FullName      string      `json:"full_name"`
	Private       bool        `json:"private"`
	DefaultBranch string      `json:"default_branch"`
	HTMLURL       string      `json:"html_url"`
	Owner         Owner       `json:"owner"`
	Permissions   Permissions `json:"permissions"`
}

// Owner is the repository owner login.
type Owner struct {
	Login string `json:"login"`
}

// Permissions are the authenticated user's permissions on the repo.
type Permissions struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

// Client calls GitHub REST APIs with a user access token.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// New creates a GitHub API client.
func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{httpClient: httpClient, baseURL: "https://api.github.com"}
}

// WithBaseURL overrides the API base (tests).
func (c *Client) WithBaseURL(base string) *Client {
	c.baseURL = base
	return c
}

// ListRepositories lists repositories the user can access (paginated first pages).
// affiliation=owner,collaborator covers owned repos and those with write access.
func (c *Client) ListRepositories(ctx context.Context, accessToken string) ([]Repository, error) {
	var all []Repository
	page := 1
	for page <= 5 { // hard cap to keep responses bounded for the assignment
		url := fmt.Sprintf("%s/user/repos?per_page=100&page=%d&affiliation=owner,collaborator&sort=updated", c.baseURL, page)
		var batch []Repository
		if err := c.getJSON(ctx, accessToken, url, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}
	return all, nil
}

// GetRepository fetches a single repository by GitHub numeric ID.
func (c *Client) GetRepository(ctx context.Context, accessToken string, githubRepoID int64) (Repository, error) {
	url := fmt.Sprintf("%s/repositories/%d", c.baseURL, githubRepoID)
	var repo Repository
	if err := c.getJSON(ctx, accessToken, url, &repo); err != nil {
		return Repository{}, err
	}
	if repo.ID == 0 {
		return Repository{}, ErrNotFound
	}
	return repo, nil
}

func (c *Client) getJSON(ctx context.Context, accessToken, url string, dest any) error {
	return c.doJSON(ctx, http.MethodGet, accessToken, url, nil, dest, []int{http.StatusOK})
}

// AddIssueLabels adds labels to an issue or pull request (PR uses the issue number).
func (c *Client) AddIssueLabels(ctx context.Context, accessToken, owner, repo string, issueNumber int, labels []string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)
	body, err := json.Marshal(map[string]any{"labels": labels})
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodPost, accessToken, url, body, nil, []int{http.StatusOK, http.StatusCreated})
}

// CreateIssueComment posts a comment on an issue or pull request.
func (c *Client) CreateIssueComment(ctx context.Context, accessToken, owner, repo string, issueNumber int, comment string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, issueNumber)
	body, err := json.Marshal(map[string]string{"body": comment})
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodPost, accessToken, url, body, nil, []int{http.StatusOK, http.StatusCreated})
}

func (c *Client) doJSON(ctx context.Context, method, accessToken, url string, body []byte, dest any, okCodes []int) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRetryable, err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("%w: read body: %v", ErrRetryable, err)
	}

	for _, code := range okCodes {
		if res.StatusCode == code {
			if dest != nil {
				if err := json.Unmarshal(respBody, dest); err != nil {
					return fmt.Errorf("decode github json: %w", err)
				}
			}
			return nil
		}
	}

	switch res.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		if res.Header.Get("X-RateLimit-Remaining") == "0" {
			return ErrRateLimited
		}
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnprocessableEntity, http.StatusConflict:
		return fmt.Errorf("%w: github status %d", ErrPermanent, res.StatusCode)
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		if res.StatusCode >= 500 {
			return fmt.Errorf("%w: github status %d", ErrRetryable, res.StatusCode)
		}
		if res.StatusCode >= 400 {
			return fmt.Errorf("%w: github status %d", ErrPermanent, res.StatusCode)
		}
		return fmt.Errorf("github status %s", strconv.Itoa(res.StatusCode))
	}
}
