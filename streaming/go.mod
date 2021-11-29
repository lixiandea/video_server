module github.com/lixiandea/video_server/streaming

go 1.16

require (
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lixiandea/video_server/entity v0.0.0-20211028101928-3af935eae33d
)


replace (
		github.com/lixiandea/video_server/entity  => ../entity
)