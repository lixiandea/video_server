// Package comments defines the comment API interface
package comments

// Comment API endpoints:
// POST /api/v1/videos/{id}/comments - Add a comment to a video (requires auth)
// GET /api/v1/videos/{id}/comments - Get comments for a video (requires auth)
// GET /api/v1/comments/{id} - Get specific comment (requires auth)
// PUT /api/v1/comments/{id} - Update a comment (requires auth)
// DELETE /api/v1/comments/{id} - Delete a comment (requires auth)