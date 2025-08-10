package githubclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

type Repo struct {
	Name          string `json:"name"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	DefaultBranch string `json:"default_branch"`
}

func New(token string) *Client {
	return &Client{httpClient: http.DefaultClient, baseURL: "https://api.github.com/", token: token}
}

// WithBaseURL overrides the default API base URL (useful for tests or GH Enterprise)
func (c *Client) WithBaseURL(base string) *Client {
	cp := *c
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	cp.baseURL = base
	return &cp
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values) (*http.Request, error) {
	u := c.baseURL + strings.TrimPrefix(path, "/")
	if len(query) > 0 {
		if strings.Contains(u, "?") {
			u += "&" + query.Encode()
		} else {
			u += "?" + query.Encode()
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github api %s: %s", req.URL.Path, resp.Status)
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// ListOrgRepos lists all repositories for an organization with pagination.
func (c *Client) ListOrgRepos(ctx context.Context, org string) ([]Repo, error) {
	var all []Repo
	page := 1
	for {
		q := url.Values{}
		q.Set("per_page", "100")
		q.Set("page", fmt.Sprintf("%d", page))
		req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/orgs/%s/repos", org), q)
		if err != nil {
			return nil, err
		}
		var repos []Repo
		if err := c.do(req, &repos); err != nil {
			return nil, err
		}
		// Append whatever we received; if zero, we finish.
		all = append(all, repos...)
		if len(repos) == 0 {
			break
		}
		page++
	}
	return all, nil
}
