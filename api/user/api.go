// Package user defines the user API interface
package user

// User API endpoints:
// POST /api/v1/users/register - Register a new user
// POST /api/v1/users/login - Login to the system
// POST /api/v1/auth/refresh - Rotate refresh cookie and issue an access token
// POST /api/v1/auth/logout - Revoke the refresh session
// GET /api/v1/users/profile - Get user profile (requires auth)
// PUT /api/v1/users/profile - Update user profile (requires auth)
// DELETE /api/v1/users/account - Delete user account (requires auth)
