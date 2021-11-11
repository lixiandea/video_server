module cmd

go 1.16

require (
	github.com/lixiandea/video_server/scheduler v0.0.0-20211108171907-7591bb14b15f
	github.com/lixiandea/video_server/streaming v0.0.0-20211108171907-7591bb14b15f
	github.com/lixiandea/video_server/user_service v0.0.0-20211108171907-7591bb14b15f
)

replace (
	github.com/lixiandea/video_server/dbops => ../dbops
	github.com/lixiandea/video_server/entity => ../entity
	github.com/lixiandea/video_server/streaming => ../streaming
	github.com/lixiandea/video_server/user_service => ../user_service
)
