package message

import "testing"

// TestUserTextWithMediaUsesFileIDs 验证图片和文件块保留服务端文件编号及 MIME 元数据。
func TestUserTextWithMediaUsesFileIDs(t *testing.T) {
	message := UserTextWithMedia(
		"分析这两个附件",
		[]string{"附件列表"},
		[]ImageData{{FileID: "file_image", MIMEType: "image/png"}},
		[]FileData{{FileID: "file_pdf", MIMEType: "application/pdf", Name: "report.pdf"}},
	)
	if len(message.ContentBlocks) != 3 {
		t.Fatalf("content block count = %d, want 3", len(message.ContentBlocks))
	}
	if message.ContentBlocks[0].UserInputText == nil || message.ContentBlocks[0].UserInputText.Text != "分析这两个附件\n\n附件列表" {
		t.Fatalf("text block = %#v, want user text and attachment details", message.ContentBlocks[0].UserInputText)
	}
	image := message.ContentBlocks[1].UserInputImage
	if image == nil || image.FileID != "file_image" || image.MIMEType != "image/png" || image.Base64Data != "" {
		t.Fatalf("image block = %#v, want file_id without base64", image)
	}
	file := message.ContentBlocks[2].UserInputFile
	if file == nil || file.FileID != "file_pdf" || file.MIMEType != "application/pdf" || file.Name != "report.pdf" {
		t.Fatalf("file block = %#v, want PDF file_id and metadata", file)
	}
}
