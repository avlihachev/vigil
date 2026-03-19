package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type UpdateInfo struct {
	Version     string `json:"version"`
	DownloadURL string `json:"downloadURL"`
}

type Updater struct {
	currentVersion string
	apiURL         string
	client         *http.Client
}

func NewUpdater(currentVersion, apiURL string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		apiURL:         apiURL,
		client:         &http.Client{Timeout: 10 * time.Second},
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func (u *Updater) Check() (*UpdateInfo, error) {
	resp, err := u.client.Get(u.apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	remote := strings.TrimPrefix(rel.TagName, "v")
	if !newerVersion(remote, u.currentVersion) {
		return nil, nil
	}

	return &UpdateInfo{
		Version:     remote,
		DownloadURL: rel.HTMLURL,
	}, nil
}

func newerVersion(a, b string) bool {
	ap := parseVersion(a)
	bp := parseVersion(b)
	for i := 0; i < 3; i++ {
		if ap[i] > bp[i] {
			return true
		}
		if ap[i] < bp[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		result[i], _ = strconv.Atoi(p)
	}
	return result
}
