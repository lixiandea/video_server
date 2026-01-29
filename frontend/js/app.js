// frontend/js/app.js

// 显示响应结果
function displayResponse(response) {
    const output = document.getElementById('responseOutput');
    output.textContent = JSON.stringify(response, null, 2);
    output.classList.add('fade-in');
    
    // 根据响应状态设置样式
    if (response.success) {
        output.style.backgroundColor = '#d4edda';
        output.style.borderColor = '#c3e6cb';
    } else {
        output.style.backgroundColor = '#f8d7da';
        output.style.borderColor = '#f5c6cb';
    }
}

// 清除响应结果
function clearResponse() {
    const output = document.getElementById('responseOutput');
    output.textContent = '请执行 API 操作以查看响应...';
    output.style.backgroundColor = '#f8f9fa';
    output.style.borderColor = '#e9ecef';
    output.classList.remove('fade-in');
}

// 用户管理功能
async function registerUser() {
    const username = document.getElementById('registerUsername').value;
    const password = document.getElementById('registerPassword').value;
    
    if (!username || !password) {
        alert('请填写用户名和密码');
        return;
    }
    
    const response = await apiClient.registerUser(username, password);
    displayResponse(response);
}

async function loginUser() {
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;
    
    if (!username || !password) {
        alert('请填写用户名和密码');
        return;
    }
    
    const response = await apiClient.loginUser(username, password);
    
    // 如果登录成功，自动填充token
    if (response.success && response.data.token) {
        document.getElementById('authToken').value = response.data.token;
        apiClient.token = response.data.token;
    }
    
    displayResponse(response);
}

async function getUserProfile() {
    const response = await apiClient.getUserProfile();
    displayResponse(response);
}

async function updateUserProfile() {
    const username = prompt('请输入新的用户名（留空则不修改）:');
    const password = prompt('请输入新的密码（留空则不修改）:');
    
    const updates = {};
    if (username) updates.username = username;
    if (password) updates.password = password;
    
    if (Object.keys(updates).length === 0) {
        alert('没有提供任何更新信息');
        return;
    }
    
    const response = await apiClient.updateUserProfile(updates);
    displayResponse(response);
}

async function deleteUserAccount() {
    if (!confirm('确定要删除您的账户吗？此操作不可撤销！')) {
        return;
    }
    
    const response = await apiClient.deleteUserAccount();
    displayResponse(response);
}

// 视频管理功能
async function uploadVideo() {
    const fileInput = document.getElementById('videoFile');
    const titleInput = document.getElementById('videoTitle');
    
    if (!fileInput.files[0]) {
        alert('请选择一个视频文件');
        return;
    }
    
    if (!titleInput.value) {
        alert('请输入视频标题');
        return;
    }
    
    // 显示上传进度
    const response = await apiClient.uploadVideo(fileInput.files[0], titleInput.value);
    displayResponse(response);
}

async function getVideoInfo() {
    const videoId = document.getElementById('videoId').value;
    
    if (!videoId) {
        alert('请输入视频 ID');
        return;
    }
    
    const response = await apiClient.getVideoInfo(videoId);
    displayResponse(response);
}

async function getUserVideos() {
    const response = await apiClient.getUserVideos();
    displayResponse(response);
}

async function deleteVideo() {
    const videoId = document.getElementById('videoId').value;
    
    if (!videoId) {
        alert('请输入视频 ID');
        return;
    }
    
    if (!confirm('确定要删除这个视频吗？')) {
        return;
    }
    
    const response = await apiClient.deleteVideo(videoId);
    displayResponse(response);
}

// 评论管理功能
async function addComment() {
    const videoId = document.getElementById('commentVideoId').value;
    const content = document.getElementById('commentContent').value;
    
    if (!videoId) {
        alert('请输入视频 ID');
        return;
    }
    
    if (!content) {
        alert('请输入评论内容');
        return;
    }
    
    const response = await apiClient.addComment(videoId, content);
    displayResponse(response);
}

async function getComments() {
    const videoId = document.getElementById('getCommentVideoId').value;
    
    if (!videoId) {
        alert('请输入视频 ID');
        return;
    }
    
    const response = await apiClient.getComments(videoId);
    displayResponse(response);
}

async function getSpecificComment() {
    const commentId = prompt('请输入评论 ID:');
    
    if (!commentId) {
        return;
    }
    
    const response = await apiClient.getSpecificComment(commentId);
    displayResponse(response);
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    console.log('视频服务测试平台已加载');
    
    // 设置API基础URL的默认值
    const baseUrlInput = document.getElementById('apiBaseUrl');
    if (baseUrlInput && !baseUrlInput.value) {
        baseUrlInput.value = 'http://localhost:8080/api/v1';
    }
});