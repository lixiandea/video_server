package defs

type UserCredential struct {
	UserName string `json:"user_name"`
	Pwd      string `json:"password"`
}

type VideoInfo struct {
	Id          string
	AuthorId    int
	Name        string
	DisplayTime string
	CreateTime  string
}

type Comment struct {
	Id       string
	AuthorId int
	VideoId  string
	Content  string
	Ctime    string
}

type SimpleSession struct {
	UserName string
	TTL      int64
}
