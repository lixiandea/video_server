// Package videos defines the video API interface
package videos

// Video API endpoints:
// POST /api/v1/videos/upload - Upload a video (requires auth)
// GET /api/v1/videos - List active videos (public)
// GET /api/v1/recommendations - Personalized recommendations when authenticated (public)
// GET /api/v1/videos/{id} - Get video information (requires auth)
// GET /api/v1/videos/{id}/related - Related recommendations (requires auth)
// GET /api/v1/admin/videos - List videos for moderation (requires admin)
// POST /api/v1/admin/videos/{id}/review - Approve or reject a video (requires admin)
// GET /api/v1/videos/{id}/stream - Stream video content (requires auth)
// GET /api/v1/users/videos - Get user's videos (requires auth)
// DELETE /api/v1/videos/{id} - Delete a video (requires auth)
