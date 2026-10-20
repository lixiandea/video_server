package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/lixiandea/video_server/internal/models"
	"github.com/lixiandea/video_server/pkg/database"
	apperrors "github.com/lixiandea/video_server/pkg/errors"
	"github.com/lixiandea/video_server/pkg/redis"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VideoService struct {
	db    *gorm.DB
	cache *redis.CacheManager
}

func NewVideoService() *VideoService {
	return &VideoService{
		db:    database.GetDB(),
		cache: redis.NewCacheManager(redis.GetClient()),
	}
}

// videoCacheKey 定义视频缓存的键名，读写两侧必须共用同一规则。
func videoCacheKey(id string) string {
	return fmt.Sprintf("video:%s", id)
}

// invalidateVideoCache 在写操作后清除缓存，避免继续返回旧数据。
// 缓存删除失败不应阻断写操作，CacheManager 内部已记录告警。
func (s *VideoService) invalidateVideoCache(videoID string) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(context.Background(), videoCacheKey(videoID))
}

func (s *VideoService) CreateVideo(authorID uint, name string) (*models.Video, error) {
	videoUUID := uuid.New().String()
	ctime := time.Now().Format("Jan 02 2006, 15:04:05")

	video := &models.Video{
		UUID:         videoUUID,
		AuthorID:     authorID,
		Name:         name,
		DisplayCTime: ctime,
		Status:       "processing",
	}

	result := s.db.Create(video)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create video: %w", result.Error)
	}

	return video, nil
}

func (s *VideoService) GetVideoByID(videoID string) (*models.Video, error) {
	ctx := context.Background()
	cacheKey := videoCacheKey(videoID)
	var video models.Video

	// 尝试从缓存获取
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &video); err == nil && video.UUID != "" {
			return &video, nil
		}
	}

	// 从数据库获取
	result := s.db.First(&video, "uuid = ?", videoID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, apperrors.NotFoundf("video not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	// 设置缓存（5 分钟过期）
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheKey, &video, 5*time.Minute)
	}

	return &video, nil
}

func (s *VideoService) GetVideosByUserID(userID uint, limit, offset int) ([]*models.Video, error) {
	var videos []*models.Video
	result := s.db.Where("author_id = ?", userID).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&videos)

	if result.Error != nil {
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	return videos, nil
}

// GetVideosByUserIDWithTotal returns videos with actual total count for pagination
func (s *VideoService) GetVideosByUserIDWithTotal(userID uint, limit, offset int) ([]*models.Video, int64, error) {
	var videos []*models.Video
	var total int64

	// Get total count
	if err := s.db.Model(&models.Video{}).Where("author_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count videos: %w", err)
	}

	// Get paginated videos
	result := s.db.Where("author_id = ?", userID).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&videos)

	if result.Error != nil {
		return nil, 0, fmt.Errorf("database error: %w", result.Error)
	}

	return videos, total, nil
}

// GetVideos 分页查询可播放（status=active）的视频，query 非空时按名称模糊匹配。
// 返回 (视频列表, 总数)，供 GET /videos 使用。
func (s *VideoService) GetVideos(limit, offset int, query string) ([]*models.Video, int64, error) {
	var videos []*models.Video
	var total int64

	base := func() *gorm.DB {
		q := s.db.Model(&models.Video{}).Where("status = ?", "active")
		if query != "" {
			q = q.Where("name LIKE ?", "%"+query+"%")
		}
		return q
	}

	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count videos: %w", err)
	}

	result := base().Limit(limit).Offset(offset).Order("created_at DESC").Find(&videos)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("database error: %w", result.Error)
	}

	return videos, total, nil
}

const (
	RecommendationModePersonalized = "personalized"
	RecommendationModePopular      = "popular"

	recommendationHistoryLimit = 100
	recommendationCandidateCap = 200
)

// VideoRecommendation 是推荐视频及其可解释理由。
type VideoRecommendation struct {
	Video  *models.Video
	Reason string
	Score  float64
}

type recommendationAffinity struct {
	VideoUUID string
	Watchers  int64
}

// RecordVideoView 以幂等方式记录最近观看。重复观看只刷新 updated_at，
// 写入失败由调用方告警，不影响视频详情展示。
func (s *VideoService) RecordVideoView(userID uint, videoID string) error {
	now := time.Now()
	view := &models.VideoView{
		UserID:    userID,
		VideoUUID: videoID,
	}
	result := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "video_uuid"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"updated_at": now,
		}),
	}).Create(view)
	if result.Error != nil {
		return fmt.Errorf("failed to record video view: %w", result.Error)
	}
	return nil
}

// GetRecommendations 聚合最近观看、协同观看与内容热度生成推荐。
// 无登录或行为不足时按热度和新鲜度退化为公开推荐。
func (s *VideoService) GetRecommendations(userID *uint, excludeVideoIDs []string, limit int) ([]VideoRecommendation, string, error) {
	if limit < 1 || limit > 50 {
		limit = 12
	}

	excluded := make(map[string]struct{}, len(excludeVideoIDs)+recommendationHistoryLimit)
	for _, id := range excludeVideoIDs {
		if id != "" {
			excluded[id] = struct{}{}
		}
	}

	affinity := make(map[string]int64)
	mode := RecommendationModePopular

	if userID != nil {
		var viewedIDs []string
		result := s.db.Model(&models.VideoView{}).
			Where("user_id = ?", *userID).
			Order("updated_at DESC").
			Limit(recommendationHistoryLimit).
			Pluck("video_uuid", &viewedIDs)
		if result.Error != nil {
			return nil, mode, fmt.Errorf("failed to load viewing history: %w", result.Error)
		}

		if len(viewedIDs) > 0 {
			mode = RecommendationModePersonalized
			for _, id := range viewedIDs {
				excluded[id] = struct{}{}
			}

			var rows []recommendationAffinity
			result = s.db.Table("video_views AS seed").
				Select("candidate.video_uuid, COUNT(DISTINCT candidate.user_id) AS watchers").
				Joins("JOIN video_views AS peer_watch ON peer_watch.video_uuid = seed.video_uuid AND peer_watch.user_id <> seed.user_id").
				Joins("JOIN video_views AS candidate ON candidate.user_id = peer_watch.user_id").
				Where("seed.user_id = ? AND seed.deleted_at IS NULL", *userID).
				Where("seed.video_uuid IN ?", viewedIDs).
				Where("peer_watch.deleted_at IS NULL AND candidate.deleted_at IS NULL").
				Where("candidate.video_uuid NOT IN ?", mapKeys(excluded)).
				Group("candidate.video_uuid").
				Order("watchers DESC").
				Limit(100).
				Scan(&rows)
			if result.Error != nil {
				return nil, mode, fmt.Errorf("failed to aggregate viewing affinity: %w", result.Error)
			}
			for _, row := range rows {
				affinity[row.VideoUUID] = row.Watchers
			}
		}
	}

	var candidates []*models.Video
	query := s.db.Where("status = ?", "active")
	if len(excluded) > 0 {
		query = query.Where("uuid NOT IN ?", mapKeys(excluded))
	}
	if userID != nil {
		query = query.Where("author_id <> ?", *userID)
	}
	result := query.
		Order("(view_count + like_count * 3) DESC").
		Order("created_at DESC").
		Limit(recommendationCandidateCap).
		Find(&candidates)
	if result.Error != nil {
		return nil, mode, fmt.Errorf("failed to load recommendation candidates: %w", result.Error)
	}

	// 协同候选可能热度较低，补充进候选集后统一排序。
	affinityIDs := make([]string, 0, len(affinity))
	for id := range affinity {
		affinityIDs = append(affinityIDs, id)
	}
	if len(affinityIDs) > 0 {
		var affinityVideos []*models.Video
		affinityQuery := s.db.Where("status = ? AND uuid IN ?", "active", affinityIDs)
		if len(excluded) > 0 {
			affinityQuery = affinityQuery.Where("uuid NOT IN ?", mapKeys(excluded))
		}
		if userID != nil {
			affinityQuery = affinityQuery.Where("author_id <> ?", *userID)
		}
		result = affinityQuery.Find(&affinityVideos)
		if result.Error != nil {
			return nil, mode, fmt.Errorf("failed to load affinity candidates: %w", result.Error)
		}
		candidates = mergeRecommendationCandidates(candidates, affinityVideos)
	}

	return rankRecommendations(candidates, affinity, limit, time.Now()), mode, nil
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func mergeRecommendationCandidates(primary, additional []*models.Video) []*models.Video {
	seen := make(map[string]struct{}, len(primary)+len(additional))
	merged := make([]*models.Video, 0, len(primary)+len(additional))
	for _, videos := range [][]*models.Video{primary, additional} {
		for _, video := range videos {
			if video == nil {
				continue
			}
			if _, ok := seen[video.UUID]; ok {
				continue
			}
			seen[video.UUID] = struct{}{}
			merged = append(merged, video)
		}
	}
	return merged
}

func rankRecommendations(videos []*models.Video, affinity map[string]int64, limit int, now time.Time) []VideoRecommendation {
	recommendations := make([]VideoRecommendation, 0, len(videos))
	for _, video := range videos {
		if video == nil || video.Status != "active" {
			continue
		}

		watchers := affinity[video.UUID]
		score := float64(watchers) * 60
		score += math.Log1p(float64(video.ViewCount)) * 8
		score += math.Log1p(float64(video.LikeCount)) * 12

		ageHours := now.Sub(video.CreatedAt).Hours()
		if ageHours < 0 {
			ageHours = 0
		}
		freshness := math.Max(0, 1-ageHours/(24*7))
		score += freshness * 10

		reason := "近期热门"
		if watchers > 0 {
			reason = "看过类似视频的人也在看"
		} else if ageHours <= 24*7 {
			reason = "新上架"
		}

		recommendations = append(recommendations, VideoRecommendation{
			Video:  video,
			Reason: reason,
			Score:  score,
		})
	}

	sort.SliceStable(recommendations, func(i, j int) bool {
		if recommendations[i].Score == recommendations[j].Score {
			return recommendations[i].Video.CreatedAt.After(recommendations[j].Video.CreatedAt)
		}
		return recommendations[i].Score > recommendations[j].Score
	})

	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}
	return recommendations
}

// GetAuthorNames 批量查询作者名，避免列表接口逐条查询作者（N+1）。
func (s *VideoService) GetAuthorNames(authorIDs []uint) (map[uint]string, error) {
	names := make(map[uint]string)
	if len(authorIDs) == 0 {
		return names, nil
	}

	var users []*models.User
	if err := s.db.Where("id IN ?", authorIDs).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	for _, u := range users {
		names[u.ID] = u.LoginName
	}

	return names, nil
}

func (s *VideoService) UpdateVideo(videoID string, updates map[string]interface{}) error {
	result := s.db.Model(&models.Video{}).Where("uuid = ?", videoID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update video: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.NotFoundf("video not found")
	}

	s.invalidateVideoCache(videoID)
	return nil
}

// SetVideoStatus 更新转码阶段状态（processing / active / failed）。
// 已删除的视频保持 deleted，避免转码结果把已删除内容重新发布。
func (s *VideoService) SetVideoStatus(videoID, status string) error {
	result := s.db.Model(&models.Video{}).
		Where("uuid = ? AND status <> ?", videoID, "deleted").
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update video status: %w", result.Error)
	}

	s.invalidateVideoCache(videoID)
	return nil
}

// IncrementViewCount 记录一次观看。
// 播放量是高频写入，用 SQL 自增避免「读-改-写」丢失更新，并让 updated_at 保持稳定。
func (s *VideoService) IncrementViewCount(videoID string) error {
	result := s.db.Model(&models.Video{}).Where("uuid = ?", videoID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1"))
	if result.Error != nil {
		return fmt.Errorf("failed to increment view count: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.NotFoundf("video not found")
	}

	s.invalidateVideoCache(videoID)
	return nil
}

func (s *VideoService) DeleteVideo(videoID string) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Video{}).Where("uuid = ?", videoID).Update("status", "deleted")
		if result.Error != nil {
			return fmt.Errorf("failed to delete video: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return apperrors.NotFoundf("video not found")
		}
		if err := tx.Unscoped().Where("video_uuid = ?", videoID).Delete(&models.VideoView{}).Error; err != nil {
			return fmt.Errorf("failed to delete video views: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.invalidateVideoCache(videoID)
	return nil
}

func (s *VideoService) HardDeleteVideo(videoID string) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("uuid = ?", videoID).Delete(&models.Video{})
		if result.Error != nil {
			return fmt.Errorf("failed to hard delete video: %w", result.Error)
		}
		if err := tx.Unscoped().Where("video_uuid = ?", videoID).Delete(&models.VideoView{}).Error; err != nil {
			return fmt.Errorf("failed to delete video views: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.invalidateVideoCache(videoID)
	return nil
}

func (s *VideoService) AddDeletionRecord(videoID, reason string) error {
	record := &models.VideoDeletionRecord{
		VideoUUID: videoID,
		Reason:    reason,
	}

	result := s.db.Create(record)
	if result.Error != nil {
		return fmt.Errorf("failed to add deletion record: %w", result.Error)
	}

	return nil
}

func (s *VideoService) GetDeletionRecords(limit int) ([]*models.VideoDeletionRecord, error) {
	var records []*models.VideoDeletionRecord
	result := s.db.Limit(limit).Order("created_at DESC").Find(&records)
	if result.Error != nil {
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	return records, nil
}

func (s *VideoService) RemoveDeletionRecord(videoID string) error {
	result := s.db.Where("video_uuid = ?", videoID).Delete(&models.VideoDeletionRecord{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove deletion record: %w", result.Error)
	}

	return nil
}

// 根据用户ID获取用户信息
func (s *VideoService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	result := s.db.First(&user, "id = ?", userID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, apperrors.NotFoundf("user not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	return &user, nil
}
