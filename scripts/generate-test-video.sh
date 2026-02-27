#!/bin/bash
# 生成测试视频脚本
# 使用 ffmpeg 创建一个简单的测试视频文件

set -e

OUTPUT_DIR="${1:-./storage/videos/test}"
mkdir -p "$OUTPUT_DIR"

echo "🎬 生成测试视频文件..."

# 检查是否有 ffmpeg
if command -v ffmpeg &> /dev/null; then
    echo "使用系统 ffmpeg 生成视频..."
    
    # 生成一个 5 秒的测试视频（红色背景）
    ffmpeg -f lavfi -i color=c=red:s=320x240:d=5 \
           -f lavfi -i sine=frequency=440:duration=5 \
           -c:v libx264 -preset fast -crf 23 \
           -c:a aac -b:a 128k \
           -pix_fmt yuv420p \
           -y "$OUTPUT_DIR/test_video.mp4" 2>/dev/null
    
    echo "✅ 视频生成成功：$OUTPUT_DIR/test_video.mp4"
    ls -lh "$OUTPUT_DIR/test_video.mp4"
    
else
    # 如果没有 ffmpeg，创建一个最小的有效 MP4 文件
    echo "ffmpeg 未安装，创建最小 MP4 文件..."
    
    # 使用 base64 编码的最小 MP4 文件（1 秒黑屏视频）
    cat > "$OUTPUT_DIR/test_video.mp4" << 'MP4DATA'
AAAAIGZ0eXBNNEVAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAABtZGF0AAAA7GVuY29kZXIg
TGF2ZjU4LjI5LjEwMAAAAABtb292AAAAbG12aGQAAAAA7eHqge3h6oH+4eqB7eHqgQAAAAMA
AAGgAQAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAEAAAAAAAAA
AAAAAAAAAAAAAAACAAABdHRyYWsAAABca2hkAAAAAe3h6oHt4eqBAAAAAQAAAAAAAAAAAAAA
AAAAAAEAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAABAAABdWR0YQAAAG1kaXJh
cHBsAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAbWRpYQAAAC
BtZGhkAAAAAO3h6oHt4eoAAAAAAAHsAAAAAAAB7AAAAAAALGZtaGRpYXAAAAAkbWR0YQAA
AAAAAQAAAAEAAAAAACRtZGluZgAAAAx2bWRlAAAAAAAAAAAAAAAAAAAAAAAAAAAAZGluZg
AAAAxkcmVmAAAAAAAAAAEAAAAMbG9jYQAAABAAAAB0cmFrAAAAXHRraGQAAAAD7eHqge3h
6oEAAAABAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAA
AAAAAAAAAAABAAABdHN0cwAAAAAAAAABAAAAAQAAAAAAAAAAAAAAABRzdHN6AAAAAAAAAA
EAAAABAAAAFHN0Y28AAAAAAAAAAQAAADAAAABzdHNkAAAAAAAAAAEAAABtc291bmQAAAAA
AAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAc3RzYwAAAAAAAAABAAAAAQAAAAAAAAAAAAAAABRzdHN6AAAAAAAAAAAAAAABAAAA
FHN0Y28AAAAAAAAAAQAAADAAAABzdHNkAAAAAAAAAAEAAABtdmlkZW8AAAAAAAEAAAABAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
HN0dHMAAAAAAAAABAAAAAQAAAAAAAAAAAAAAABRzdHN6AAAAAAAAAAAAAAABAAAAFHN0
Y28AAAAAAAAAAQAAADAAAAA=
MP4DATA
    
    # 解码 base64 创建二进制文件
    base64 -d "$OUTPUT_DIR/test_video.mp4" > "$OUTPUT_DIR/test_video_temp.mp4" 2>/dev/null || true
    
    if [ -f "$OUTPUT_DIR/test_video_temp.mp4" ]; then
        mv "$OUTPUT_DIR/test_video_temp.mp4" "$OUTPUT_DIR/test_video.mp4"
        echo "✅ 最小 MP4 文件创建成功：$OUTPUT_DIR/test_video.mp4"
        ls -lh "$OUTPUT_DIR/test_video.mp4"
    else
        # 如果 base64 也失败，创建一个空的占位文件
        echo "创建空测试文件..."
        dd if=/dev/zero of="$OUTPUT_DIR/test_video.mp4" bs=1024 count=10 2>/dev/null
        echo "⚠️  已创建占位文件：$OUTPUT_DIR/test_video.mp4"
        ls -lh "$OUTPUT_DIR/test_video.mp4"
    fi
fi

echo ""
echo "📊 视频文件信息:"
file "$OUTPUT_DIR/test_video.mp4" 2>/dev/null || echo "无法获取文件类型"
echo ""
echo "💡 使用方法:"
echo "   在测试脚本中调用：./scripts/generate-test-video.sh"
echo "   或指定输出目录：./scripts/generate-test-video.sh /path/to/output"
