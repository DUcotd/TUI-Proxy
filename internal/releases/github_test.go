package releases

import (
	"testing"

	"clashctl/internal/system"
)

func TestSelectGitHubReleasePrefersStableByDefault(t *testing.T) {
	releases := []GitHubRelease{
		{TagName: "v3.0.0-rc1", Prerelease: true},
		{TagName: "v2.9.0", Prerelease: false},
	}

	got := SelectGitHubRelease(releases, false)
	if got == nil {
		t.Fatal("SelectGitHubRelease() returned nil")
	}
	if got.TagName != "v2.9.0" {
		t.Fatalf("SelectGitHubRelease() = %q, want v2.9.0", got.TagName)
	}
}

func TestSelectGitHubReleaseIncludesPrereleaseWhenRequested(t *testing.T) {
	releases := []GitHubRelease{
		{TagName: "v3.0.0-rc1", Prerelease: true},
		{TagName: "v2.9.0", Prerelease: false},
	}

	got := SelectGitHubRelease(releases, true)
	if got == nil {
		t.Fatal("SelectGitHubRelease() returned nil")
	}
	if got.TagName != "v3.0.0-rc1" {
		t.Fatalf("SelectGitHubRelease() = %q, want v3.0.0-rc1", got.TagName)
	}
}

func TestFindGitHubReleaseAsset(t *testing.T) {
	release := &GitHubRelease{Assets: []GitHubAsset{{Name: "clashctl-linux-amd64", BrowserDownloadURL: "https://example.com/a"}}}

	asset, ok := FindGitHubReleaseAsset(release, "clashctl-linux-amd64")
	if !ok {
		t.Fatal("FindGitHubReleaseAsset() should find requested asset")
	}
	if asset.BrowserDownloadURL != "https://example.com/a" {
		t.Fatalf("asset = %#v", asset)
	}
}

func TestNamedDownloads(t *testing.T) {
	release := &GitHubRelease{Assets: []GitHubAsset{{Name: "clashctl-linux-amd64", BrowserDownloadURL: "https://example.com/a"}}}

	got := NamedDownloads(release)
	want := []system.NamedDownload{{Name: "clashctl-linux-amd64", URL: "https://example.com/a"}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("NamedDownloads() = %#v, want %#v", got, want)
	}
}
