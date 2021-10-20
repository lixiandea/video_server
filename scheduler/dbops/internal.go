package dbops

import (
	"log"
)

func ReadVideoDeletionRecord(count int) ([]string, error) {
	stmtOut, err := conn.Prepare("SELECT video_id FROM video_del_rec limit ?")
	var ids []string
	if err != nil {
		return ids, err
	}

	rows, err := stmtOut.Query(count)

	if err != nil {
		log.Printf("Query video deletion record error : %v", err)
		return ids, err
	}

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}

	defer stmtOut.Close()

	return ids, err
}

func DelVideoDeletionRecord(vid string) error {
	stmtIn, err := conn.Prepare("DELETE FROM video_del_rec where id =  ?")
	if err != nil {
		return err
	}

	_, err = stmtIn.Exec(vid)

	if err != nil {
		log.Printf("Query video deletion record error : %v", err)
		return err
	}
	return nil
}
