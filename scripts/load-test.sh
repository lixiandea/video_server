#!/bin/bash
# 转码链路压测：并发上传 N 个视频，等待全部转码完成，统计吞吐、延迟与队列积压。
#
# 用法：
#   bash scripts/load-test.sh
#   UPLOADS=20 CONCURRENCY=5 bash scripts/load-test.sh
#   API_URL=http://localhost:8080 TEST_VIDEO_SRC=testdata/sample.mp4 bash scripts/load-test.sh
#
# 依赖：curl、jq；需要已登录的 API（默认 http://localhost:8080）。
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
UPLOADS="${UPLOADS:-10}"
CONCURRENCY="${CONCURRENCY:-4}"
TEST_VIDEO_SRC="${TEST_VIDEO_SRC:-$(dirname "$0")/../testdata/sample.mp4}"
POLL_INTERVAL="${POLL_INTERVAL:-2}"
TIMEOUT="${TIMEOUT:-300}"

if [ ! -f "$TEST_VIDEO_SRC" ]; then
    echo "测试视频不存在: $TEST_VIDEO_SRC" >&2
    exit 1
fi
command -v jq >/dev/null || { echo "需要 jq" >&2; exit 1; }

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

echo "=== 转码链路压测 ==="
echo "API: $API_URL  上传数: $UPLOADS  并发: $CONCURRENCY"

USERNAME="loadtest_$(date +%s)"
REG=$(curl -sS -X POST "$API_URL/api/v1/users/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$USERNAME\",\"password\":\"LoadTest123\"}")
TOKEN=$(echo "$REG" | jq -r '.token // empty')
if [ -z "$TOKEN" ]; then
    TOKEN=$(curl -sS -X POST "$API_URL/api/v1/users/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$USERNAME\",\"password\":\"LoadTest123\"}" | jq -r '.token // empty')
fi
if [ -z "$TOKEN" ]; then
    echo "注册/登录失败" >&2
    exit 1
fi
echo "测试账号: $USERNAME"

upload_one() {
    local idx=$1
    local body="$WORKDIR/upload_$idx.json"
    local meta
    meta=$(curl -sS -o "$body" -w '%{http_code} %{time_total}' \
        -X POST "$API_URL/api/v1/videos/upload" \
        -H "Authorization: Bearer $TOKEN" \
        -F "file=@$TEST_VIDEO_SRC" \
        -F "name=loadtest-$idx")
    local code=${meta%% *}
    local secs=${meta##* }
    local vid
    vid=$(jq -r '.video_id // empty' "$body" 2>/dev/null || true)
    echo "$code $secs $vid" > "$WORKDIR/result_$idx.txt"
}

start=$SECONDS
for i in $(seq 1 "$UPLOADS"); do
    upload_one "$i" &
    while [ "$(jobs -rp | wc -l)" -ge "$CONCURRENCY" ]; do
        sleep 0.05
    done
done
wait
upload_secs=$((SECONDS - start))

ok=$(grep -l '^201 ' "$WORKDIR"/result_*.txt 2>/dev/null | wc -l | tr -d ' ' || true)
fail=$((UPLOADS - ok))
latencies=$(awk '{print $2}' "$WORKDIR"/result_*.txt 2>/dev/null | sort -n || true)
if [ -n "$latencies" ]; then
    p50=$(echo "$latencies" | awk '{a[NR]=$1} END {printf "%.3f", a[int((NR+1)*0.5)]}')
    p95=$(echo "$latencies" | awk '{a[NR]=$1} END {printf "%.3f", a[int((NR+1)*0.95)]}')
else
    p50="n/a"
    p95="n/a"
fi

echo "上传成功: ${ok}/${UPLOADS}（失败 ${fail}）  耗时: ${upload_secs}s  p50=${p50}s p95=${p95}s"

deadline=$((SECONDS + TIMEOUT))
peak_pending=0
failed_videos=0
processing_videos=0
while [ "$SECONDS" -lt "$deadline" ]; do
    stats=$(curl -sS "$API_URL/api/v1/transcode/queue/stats" -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo '{}')
    pending=$(echo "$stats" | jq -r '.pending_count // 0')
    if [ "$pending" -gt "$peak_pending" ]; then
        peak_pending=$pending
    fi

    videos=$(curl -sS "$API_URL/api/v1/users/videos?page=1&limit=100" -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo '{}')
    processing_videos=$(echo "$videos" | jq '[.videos[] | select(.status == "processing")] | length' 2>/dev/null || echo 0)
    failed_videos=$(echo "$videos" | jq '[.videos[] | select(.status == "failed")] | length' 2>/dev/null || echo 0)
    if [ "$processing_videos" -eq 0 ]; then
        break
    fi
    sleep "$POLL_INTERVAL"
done
total_secs=$((SECONDS - start))

echo "转码完成耗时: ${total_secs}s（总耗时）  峰值 pending: $peak_pending  失败视频: $failed_videos"

if [ "$fail" -gt 0 ] || [ "$failed_videos" -gt 0 ] || [ "$processing_videos" -gt 0 ]; then
    echo "结果: FAIL（仍有未完成或失败的上传）"
    exit 1
fi
echo "结果: PASS"
