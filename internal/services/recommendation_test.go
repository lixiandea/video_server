package services

import (
	"testing"
	"time"

	"github.com/lixiandea/video_server/internal/models"
)

func TestRankRecommendationsPrefersAffinityAndExplainsReason(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	videos := []*models.Video{
		{
			BaseModel: models.BaseModel{
				CreatedAt: now.Add(-30 * 24 * time.Hour),
			},
			UUID:      "popular",
			Status:    "active",
			ViewCount: 5000,
			LikeCount: 100,
		},
		{
			BaseModel: models.BaseModel{
				CreatedAt: now.Add(-30 * 24 * time.Hour),
			},
			UUID:      "similar",
			Status:    "active",
			ViewCount: 10,
			LikeCount: 1,
		},
	}

	got := rankRecommendations(videos, map[string]int64{"similar": 3}, 2, now)

	if len(got) != 2 {
		t.Fatalf("expected 2 recommendations, got %d", len(got))
	}
	if got[0].Video.UUID != "similar" {
		t.Fatalf("expected affinity video first, got %q", got[0].Video.UUID)
	}
	if got[0].Reason != "看过类似视频的人也在看" {
		t.Fatalf("unexpected affinity reason: %q", got[0].Reason)
	}
	if got[1].Reason != "近期热门" {
		t.Fatalf("unexpected fallback reason: %q", got[1].Reason)
	}
}

func TestRankRecommendationsFiltersInactiveAndLimits(t *testing.T) {
	now := time.Now()
	videos := []*models.Video{
		{BaseModel: models.BaseModel{CreatedAt: now}, UUID: "active-1", Status: "active"},
		{BaseModel: models.BaseModel{CreatedAt: now}, UUID: "processing", Status: "processing"},
		{BaseModel: models.BaseModel{CreatedAt: now.Add(-time.Hour)}, UUID: "active-2", Status: "active"},
	}

	got := rankRecommendations(videos, nil, 1, now)

	if len(got) != 1 {
		t.Fatalf("expected one limited recommendation, got %d", len(got))
	}
	if got[0].Video.UUID != "active-1" {
		t.Fatalf("expected newest active video, got %q", got[0].Video.UUID)
	}
}

func TestMergeRecommendationCandidatesDeduplicatesUUID(t *testing.T) {
	first := []*models.Video{{UUID: "v1"}, {UUID: "v2"}}
	second := []*models.Video{{UUID: "v2"}, {UUID: "v3"}, nil}

	got := mergeRecommendationCandidates(first, second)

	if len(got) != 3 {
		t.Fatalf("expected 3 unique videos, got %d", len(got))
	}
}
