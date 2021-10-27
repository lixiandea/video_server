package taskrunner

import (
	"errors"
	"log"
	"os"
	"scheduler/dbops"
	"sync"
)

func deleteVideo(vid string) error {
	err := os.Remove(VIDEO_PATH + vid)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("error delete files")
		return err
	}
	return nil
}

func VideoClearDispatcher(dc dataChannel) error {
	res, err := dbops.ReadVideoDeletionRecord(DELETE_PER)

	if err != nil {
		log.Printf("Video clear dispatch error : %v", err)
		return err
	}

	if len(res) == 0 {
		return errors.New("All task finished")
	}
	for _, id := range res {
		dc <- id
	}

	return nil
}

func VideoClearExecutor(dc dataChannel) error {
	errMap := &sync.Map{}
forloop:
	for {
		select {
		case vid := <-dc:
			go func(id interface{}) {
				if err := deleteVideo(id.(string)); err != nil {
					errMap.Store(id, err)
					return
				}
				if err := dbops.DelVideoDeletionRecord(id.(string)); err != nil {
					errMap.Store(id, err)
					return
				}
			}(vid)
		default:
			break forloop
		}
	}
	var err error
	errMap.Range(func(key, value interface{}) bool {
		err = value.(error)
		if err != nil {
			return false
		}
		return true
	})
	return err
}
