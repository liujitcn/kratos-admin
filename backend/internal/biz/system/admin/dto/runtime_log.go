package dto

// RuntimeLogCursor 保存历史日志向前翻页所需的不透明状态。
type RuntimeLogCursor struct {
	FileID     string `json:"file_id"`
	Identity   string `json:"identity"`
	FilterHash string `json:"filter_hash"`
	Size       int64  `json:"size"`
	End        int64  `json:"end"`
}

// RuntimeLogDownload 保存历史日志原文件下载内容。
type RuntimeLogDownload struct {
	Name        string
	ContentType string
	Data        []byte
}
