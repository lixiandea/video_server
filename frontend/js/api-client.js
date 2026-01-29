// frontend/js/api-client.js

class ApiClient {
    constructor() {
        this.baseUrl = document.getElementById('apiBaseUrl')?.value || 'http://localhost:8080/api/v1';
        this.token = document.getElementById('authToken')?.value || '';
    }

    // 更新基本URL和token
    updateConfig() {
        this.baseUrl = document.getElementById('apiBaseUrl')?.value || 'http://localhost:8080/api/v1';
        this.token = document.getElementById('authToken')?.value || '';
    }

    // 设置请求头
    getHeaders(includeAuth = true) {
        const headers = {
            'Content-Type': 'application/json'
        };
        
        if (includeAuth && this.token) {
            headers['Authorization'] = `Bearer ${this.token}`;
        }
        
        return headers;
    }

    // 通用请求方法
    async request(endpoint, options = {}) {
        this.updateConfig();
        
        const url = `${this.baseUrl}${endpoint}`;
        const config = {
            headers: { ...this.getHeaders(options.includeAuth !== false) },
            ...options
        };

        try {
            const response = await fetch(url, config);
            const data = await response.json();
            
            return {
                success: response.ok,
                status: response.status,
                data: data,
                headers: response.headers
            };
        } catch (error) {
            return {
                success: false,
                error: error.message
            };
        }
    }

    // 用户相关API
    async registerUser(username, password) {
        return this.request('/users/register', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
            includeAuth: false
        });
    }

    async loginUser(username, password) {
        return this.request('/users/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
            includeAuth: false
        });
    }

    async getUserProfile() {
        return this.request('/users/profile', {
            method: 'GET'
        });
    }

    async updateUserProfile(updates) {
        return this.request('/users/profile', {
            method: 'PUT',
            body: JSON.stringify(updates)
        });
    }

    async deleteUserAccount() {
        return this.request('/users/account', {
            method: 'DELETE'
        });
    }

    // 视频相关API
    async uploadVideo(file, title) {
        this.updateConfig();
        
        const formData = new FormData();
        formData.append('file', file);
        formData.append('name', title);
        
        const url = `${this.baseUrl}/videos/upload`;
        
        try {
            const response = await fetch(url, {
                method: 'POST',
                body: formData,
                headers: this.token ? { 'Authorization': `Bearer ${this.token}` } : {}
            });
            
            const data = await response.json();
            
            return {
                success: response.ok,
                status: response.status,
                data: data
            };
        } catch (error) {
            return {
                success: false,
                error: error.message
            };
        }
    }

    async getVideoInfo(videoId) {
        return this.request(`/videos/${videoId}`, {
            method: 'GET'
        });
    }

    async getVideoStream(videoId) {
        this.updateConfig();
        const url = `${this.baseUrl}/videos/${videoId}/stream`;
        
        const headers = this.token ? { 'Authorization': `Bearer ${this.token}` } : {};
        
        return fetch(url, { headers });
    }

    async getUserVideos(page = 1, limit = 10) {
        return this.request(`/users/videos?page=${page}&limit=${limit}`, {
            method: 'GET'
        });
    }

    async deleteVideo(videoId) {
        return this.request(`/videos/${videoId}`, {
            method: 'DELETE'
        });
    }

    // 评论相关API
    async addComment(videoId, content) {
        return this.request(`/videos/${videoId}/comments`, {
            method: 'POST',
            body: JSON.stringify({ content })
        });
    }

    async getComments(videoId, page = 1, limit = 10) {
        return this.request(`/videos/${videoId}/comments?page=${page}&limit=${limit}`, {
            method: 'GET'
        });
    }

    async getSpecificComment(commentId) {
        return this.request(`/comments/${commentId}`, {
            method: 'GET'
        });
    }

    async updateComment(commentId, content) {
        return this.request(`/comments/${commentId}`, {
            method: 'PUT',
            body: JSON.stringify({ content })
        });
    }

    async deleteComment(commentId) {
        return this.request(`/comments/${commentId}`, {
            method: 'DELETE'
        });
    }
}

// 创建全局API客户端实例
const apiClient = new ApiClient();