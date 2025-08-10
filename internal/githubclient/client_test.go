package githubclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListOrgReposPagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/orgs/testorg/repos", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "1", "":
			_ = json.NewEncoder(w).Encode([]Repo{
				{Name: "repo1", CloneURL: "https://example.com/repo1.git", SSHURL: "git@example.com:repo1.git", DefaultBranch: "main"},
				{Name: "repo2", CloneURL: "https://example.com/repo2.git", SSHURL: "git@example.com:repo2.git", DefaultBranch: "main"},
			})
		case "2":
			_ = json.NewEncoder(w).Encode([]Repo{
				{Name: "repo3", CloneURL: "https://example.com/repo3.git", SSHURL: "git@example.com:repo3.git", DefaultBranch: "main"},
			})
		default:
			_ = json.NewEncoder(w).Encode([]Repo{})
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New("").WithBaseURL(srv.URL + "/")
	repos, err := c.ListOrgRepos(context.Background(), "testorg")
	if err != nil {
		t.Fatalf("ListOrgRepos error: %v", err)
	}
	if len(repos) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(repos))
	}
	if repos[0].Name != "repo1" || repos[2].Name != "repo3" {
		t.Fatalf("unexpected repo order or names: %+v", repos)
	}
}
