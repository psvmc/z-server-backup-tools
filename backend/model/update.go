package model

type UpdateCheckResult struct {
	Available      bool   `json:"available"`
	Enabled        bool   `json:"enabled"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseName    string `json:"releaseName"`
	Notes          string `json:"notes"`
	ReleaseURL     string `json:"releaseURL"`
}
