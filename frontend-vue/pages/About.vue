<template>
  <div class="about-page">
    <section class="hero">
      <h1>关于我们</h1>
      <p>欢迎来到 Vue Video，一个现代化的视频分享平台</p>
    </section>

    <section class="content">
      <div class="feature-section">
        <h2>我们的使命</h2>
        <p>我们致力于为用户提供高质量的视频内容和流畅的观看体验。通过技术创新和优质内容，我们希望连接创作者和观众，打造一个活跃的视频社区。</p>
      </div>

      <div class="feature-section">
        <h2>我们的特色</h2>
        <div class="features-grid">
          <div class="feature-item">
            <h3>高清播放</h3>
            <p>支持多种分辨率的视频播放，确保最佳观看体验</p>
          </div>
          <div class="feature-item">
            <h3>智能推荐</h3>
            <p>基于用户兴趣的个性化视频推荐系统</p>
          </div>
          <div class="feature-item">
            <h3>社区互动</h3>
            <p>丰富的评论和社交功能，增强用户互动</p>
          </div>
          <div class="feature-item">
            <h3>多平台支持</h3>
            <p>响应式设计，支持各种设备访问</p>
          </div>
        </div>
      </div>

      <div class="feature-section">
        <h2>联系我们</h2>
        <p>如果您有任何问题或建议，请随时通过以下方式联系我们：</p>
        <ul class="contact-list">
          <li>邮箱: contact@vuevideo.example.com</li>
          <li>客服热线: 400-123-4567</li>
          <li>工作时间: 周一至周五 9:00-18:00</li>
        </ul>
      </div>
    </section>
  </div>
</template>

<script>
export default {
  name: 'About',
  // 在服务端渲染时设置页面标题等信息
  metaInfo: {
    title: '关于我们 - Vue Video',
    meta: [
      { name: 'description', content: '了解 Vue Video 平台的使命和特色' }
    ]
  }
}
</script>

<style scoped>
.about-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.hero {
  text-align: center;
  padding: 3rem 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  margin-bottom: 2rem;
  border-radius: 8px;
}

.hero h1 {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.hero p {
  font-size: 1.2rem;
}

.content {
  margin-top: 2rem;
}

.feature-section {
  margin-bottom: 3rem;
}

.feature-section h2 {
  font-size: 1.8rem;
  margin-bottom: 1.5rem;
  color: #2c3e50;
}

.feature-section p {
  font-size: 1.1rem;
  line-height: 1.6;
  color: #555;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
  margin-top: 1.5rem;
}

.feature-item {
  background: #f8f9fa;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  transition: transform 0.3s;
}

.feature-item:hover {
  transform: translateY(-5px);
}

.feature-item h3 {
  font-size: 1.3rem;
  margin-bottom: 0.5rem;
  color: #2c3e50;
}

.contact-list {
  margin-top: 1rem;
  padding-left: 1.5rem;
}

.contact-list li {
  margin-bottom: 0.5rem;
  font-size: 1rem;
  color: #555;
}
</style>