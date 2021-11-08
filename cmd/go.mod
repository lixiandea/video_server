module cmd

go 1.16

require (
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lixiandea/video_server/scheduler v0.0.0-20211028131247-9b5df3ed3bfb
	github.com/lixiandea/video_server/streaming v0.0.0-20211028131247-9b5df3ed3bfb
	github.com/lixiandea/video_server/user_service v0.0.0-20211028131247-9b5df3ed3bfb
)

replace (
	// github.com/lixiandea/video_server/dbops v0.0.0-00010101000000-000000000000 => ../dbops
	github.com/lixiandea/video_server/entity v0.0.0-20211107142625-9324bc0eb0d1 => ../entity
	github.com/lixiandea/video_server/scheduler v0.0.0-20211028131247-9b5df3ed3bfb => ../scheduler
	github.com/lixiandea/video_server/streaming v0.0.0-20211028131247-9b5df3ed3bfb => ../streaming
	github.com/lixiandea/video_server/user_service v0.0.0-20211028131247-9b5df3ed3bfb => ../user_service
)
