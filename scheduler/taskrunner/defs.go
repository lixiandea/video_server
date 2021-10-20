package taskrunner

const (
	READY_TO_DISPATCH = "d"
	READY_TO_EXECUTE  = "e"
	CLOSE             = "c"
	DELETE_PER        = 3
	VIDEO_PATH        = "../streaming/videos/"
)

type controlChannel chan string
type dataChannel chan interface{}

type fn func(dc dataChannel) error
