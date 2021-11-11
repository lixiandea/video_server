module github.com/lixiandea/video_server/web

go 1.16

require (
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lixiandea/video_server/entity v0.0.0-20211108171907-7591bb14b15f
	github.com/lixiandea/video_server/user_service v0.0.0-20211107142625-9324bc0eb0d1

)

replace (
	github.com/lixiandea/video_server/dbops => ../dbops
	github.com/lixiandea/video_server/user_service => ../user_service
)
