module github.com/lixiandea/video_server/user_service

go 1.16

require (
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lixiandea/video_server/dbops v0.0.0-20211107142625-9324bc0eb0d1
	github.com/lixiandea/video_server/entity v0.0.0-20211107142625-9324bc0eb0d1
	github.com/satori/go.uuid v1.2.0
)

replace (
	github.com/lixiandea/video_server/dbops  => ../dbops
	github.com/lixiandea/video_server/entity  => ../entity
)
