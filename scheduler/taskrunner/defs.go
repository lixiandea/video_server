package taskrunner

const (
	READY_TO_DISPATCH = "d"
	READY_TO_EXECUTE  = "e"
	CLOSE             = "c"
)

type controlChannel chan string
type dataChannel chan interface{}

type fn func(dc dataChannel) error
