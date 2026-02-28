<template>
  <div class="upload-page">
    <div class="container">
      <div class="row justify-content-center">
        <div class="col-lg-8">
          <div class="card shadow">
            <div class="card-body">
              <h2 class="mb-4">上传视频</h2>
              
              <div v-if="error" class="alert alert-danger" role="alert">
                <i class="bi bi-exclamation-triangle"></i> {{ error }}
              </div>
              
              <div v-if="success" class="alert alert-success" role="alert">
                <i class="bi bi-check-circle"></i> 上传成功！
                <router-link :to="`/video/${uploadedVideoId}`" class="alert-link">
                  立即查看
                </router-link>
              </div>
              
              <form @submit.prevent="handleUpload">
                <div class="mb-3">
                  <label for="title" class="form-label">视频标题</label>
                  <input 
                    type="text" 
                    class="form-control" 
                    id="title" 
                    v-model="form.title"
                    required
                    placeholder="请输入视频标题"
                    maxlength="100"
                  >
                </div>
                
                <div class="mb-3">
                  <label for="description" class="form-label">视频简介</label>
                  <textarea 
                    class="form-control" 
                    id="description" 
                    v-model="form.description"
                    rows="4"
                    placeholder="简单介绍一下你的视频..."
                    maxlength="500"
                  ></textarea>
                </div>
                
                <div class="mb-3">
                  <label for="videoFile" class="form-label">视频文件</label>
                  <input 
                    type="file" 
                    class="form-control" 
                    id="videoFile" 
                    @change="handleFileChange"
                    accept="video/*"
                    required
                  >
                  <div class="form-text">
                    支持 MP4、WebM、OGG 等格式，最大支持 2GB 文件
                  </div>
                </div>
                
                <div v-if="uploading" class="mb-3">
                  <label class="form-label">上传进度</label>
                  <div class="progress">
                    <div 
                      class="progress-bar progress-bar-striped progress-bar-animated" 
                      role="progressbar"
                      :style="{ width: uploadProgress + '%' }"
                      :aria-valuenow="uploadProgress"
                      aria-valuemin="0"
                      aria-valuemax="100"
                    >
                      {{ uploadProgress }}%
                    </div>
                  </div>
                </div>
                
                <div class="d-flex gap-2">
                  <button type="submit" class="btn btn-primary" :disabled="uploading || !selectedFile">
                    <span v-if="!uploading">
                      <i class="bi bi-upload"></i> 上传视频
                    </span>
                    <span v-else>
                      <span class="spinner-border spinner-border-sm" role="status"></span>
                      上传中...
                    </span>
                  </button>
                  <router-link to="/videos" class="btn btn-outline-secondary">
                    取消
                  </router-link>
                </div>
              </form>
            </div>
          </div>
          
          <div class="alert alert-info mt-4">
            <h5><i class="bi bi-info-circle"></i> 上传说明</h5>
            <ul class="mb-0">
              <li>支持常见视频格式：MP4、WebM、OGG、MOV、AVI 等</li>
              <li>最大文件大小：2GB</li>
              <li>上传后系统将自动进行转码处理</li>
              <li>转码完成后即可公开观看</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useVideoStore } from '@/stores/video'
import videoApi from '@/api/video'

const router = useRouter()
const videoStore = useVideoStore()

const form = reactive({
  title: '',
  description: ''
})

const selectedFile = ref(null)
const uploading = ref(false)
const uploadProgress = ref(0)
const error = ref(null)
const success = ref(false)
const uploadedVideoId = ref(null)

function handleFileChange(event) {
  const file = event.target.files[0]
  if (file) {
    // 检查文件大小 (2GB limit)
    if (file.size > 2 * 1024 * 1024 * 1024) {
      error.value = '文件大小不能超过 2GB'
      selectedFile.value = null
      return
    }
    
    // 检查文件类型
    if (!file.type.startsWith('video/')) {
      error.value = '请上传视频文件'
      selectedFile.value = null
      return
    }
    
    error.value = null
    selectedFile.value = file
  }
}

async function handleUpload() {
  if (!selectedFile.value || !form.title) {
    error.value = '请填写标题并选择视频文件'
    return
  }
  
  uploading.value = true
  error.value = null
  success.value = false
  uploadProgress.value = 0
  
  const formData = new FormData()
  formData.append('title', form.title)
  formData.append('description', form.description)
  formData.append('video', selectedFile.value)
  
  try {
    const response = await videoStore.uploadVideo(formData, (progress) => {
      uploadProgress.value = progress
    })
    
    success.value = true
    uploadedVideoId.value = response.data?.id || response.data?.video_id
    
    // 2 秒后跳转到视频详情页
    setTimeout(() => {
      router.push(`/video/${uploadedVideoId.value}`)
    }, 2000)
  } catch (err) {
    error.value = err.response?.data?.message || '上传失败，请重试'
  } finally {
    uploading.value = false
  }
}
</script>

<style lang="scss" scoped>
.upload-page {
  min-height: calc(100vh - 120px);
  padding: 2rem 0;
  background: #f8f9fa;
}

.card {
  border: none;
  border-radius: 12px;
}

.progress {
  height: 25px;
}
</style>
