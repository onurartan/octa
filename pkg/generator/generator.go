package generator

import (
	"encoding/json"
	"fmt"
	"time"
	"net/http"

	"octa/pkg/generator/styles"
)

type GithubUser struct {
	Name string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	
}

func ImageResponse(name string, w http.ResponseWriter, r *http.Request) {
	style := r.URL.Query().Get("style")
	if style == "" {
		style = "initials"
	}

	switch style {
	case "initials":
		styles.GenerateInitialsAvatar(name, w, r)
	default:
		styles.GenerateInitialsAvatar(name, w, r)
	}
}

func FetchGitHubName(username string) (*GithubUser, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)
	fmt.Println(url)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "octa-app")

	 client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while fetching GitHub user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return nil, fmt.Errorf("GitHub API rate limit exceeded")
		}
	 return nil, fmt.Errorf("GitHub API status: %d", resp.StatusCode)
	}

	var user GithubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("error parsing GitHub response: %v", err)
	}

	if user.Name == "" {
		user.Name = username
	}

	return &user, nil
}
