module github.com/lixiandea/video_server/scheduler

go 1.16

require (
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lixiandea/video_server/dbops v0.0.0-20211028104229-84d994e95a45
)

replace (
	github.com/lixiandea/video_server/dbops => ../dbops
	github.com/lixiandea/video_server/entity => ../entity
	github.com/lixiandea/video_server/streaming => ../streaming
	github.com/lixiandea/video_server/user_service => ../user_service
)
