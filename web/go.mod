module github.com/lixiandea/video_server/web

go 1.17

require (
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lixiandea/video_server/entity v0.0.0-20211107142625-9324bc0eb0d1
	github.com/lixiandea/video_server/user_service v0.0.0-20211107142625-9324bc0eb0d1

)

require (
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/lixiandea/video_server/dbops v0.0.0-20211028101928-3af935eae33d // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
)


replace (
	github.com/lixiandea/video_server/user_service => ../user_service
	github.com/lixiandea/video_server/dbops => ../dbops
)