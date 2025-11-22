package util

import (
	"context"
	"errors"
	"fmt"
	"forge/pkg/log/zlog"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/unidoc/unioffice/v2/document"
	"github.com/unidoc/unioffice/v2/presentation"
	"github.com/unidoc/unipdf/v4/extractor"
	"github.com/unidoc/unipdf/v4/model"
)

// 定义MIME类型常量
const (
	mimeTypePDF  = "application/pdf"
	mimeTypeDoc  = "application/msword"
	mimeTypeDocx = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	mimeTypePPT  = "application/vnd.ms-powerpoint"
	mimeTypePPTx = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	mimeTypeZip  = "application/zip" // .docx, .pptx 实际上是ZIP格式
)

// 支持的文件扩展名
var supportedExtensions = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".ppt":  true,
	".pptx": true,
}

func ParseFile(ctx context.Context, fh *multipart.FileHeader) (text string, err error) {
	// 首先检查文件扩展名
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !supportedExtensions[ext] {
		return "", fmt.Errorf("unsupported file extension: %s", ext)
	}
	mime, err := fileHeaderMime(fh)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		zlog.CtxErrorf(ctx, "failed to detect MIME type for file %s: %v", fh.Filename, err)
		return "", err
	}

	// 根据MIME类型和文件扩展名确定文件类型
	fileType, err := determineFileType(mime, ext)
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to determine file type for %s: %v", fh.Filename, err)
		return "", err
	}

	switch fileType {
	case mimeTypePDF:
		text, err = extractPDF(fh)
	case mimeTypeDoc, mimeTypeDocx:
		text, err = extractWord(fh)
	case mimeTypePPT, mimeTypePPTx:
		text, err = extractPPT(fh)
	default:
		err = fmt.Errorf("unsupported file type: %s", fileType)
	}

	if err != nil {
		zlog.CtxErrorf(ctx, "failed to extract content from %s: %v", fh.Filename, err)
		return "", err
	}

	return text, nil
}

// 根据MIME类型和文件扩展名确定最终的文件类型
func determineFileType(mime, ext string) (string, error) {
	switch mime {
	case mimeTypePDF:
		return mimeTypePDF, nil
	case mimeTypeDoc, mimeTypeDocx:
		return mime, nil
	case mimeTypePPT, mimeTypePPTx:
		return mime, nil
	case mimeTypeZip:
		// .docx 和 .pptx 实际上是ZIP格式，需要根据扩展名进一步判断
		switch ext {
		case ".docx":
			return mimeTypeDocx, nil
		case ".pptx":
			return mimeTypePPTx, nil
		default:
			return "", fmt.Errorf("unsupported ZIP-based file format: %s", ext)
		}
	default:
		// 如果MIME类型检测失败，回退到扩展名判断
		switch ext {
		case ".pdf":
			return mimeTypePDF, nil
		case ".doc":
			return mimeTypeDoc, nil
		case ".docx":
			return mimeTypeDocx, nil
		case ".ppt":
			return mimeTypePPT, nil
		case ".pptx":
			return mimeTypePPTx, nil
		default:
			return "", fmt.Errorf("unable to determine file type: MIME=%s, ext=%s", mime, ext)
		}
	}
}

// 返回检测到的MIME类型
func fileHeaderMime(fh *multipart.FileHeader) (string, error) {
	file, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 读取文件头进行MIME类型检测
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}

	if n == 0 {
		return "", errors.New("file is empty")
	}

	return http.DetectContentType(buf[:n]), nil
}

func extractPDF(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}

	defer f.Close()

	pdfReader, err := model.NewPdfReader(f)
	if err != nil {
		return "", err
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", err
	}

	var textBuilder strings.Builder

	for i := 0; i < numPages; i++ {
		pageNum := i + 1

		page, err := pdfReader.GetPage(pageNum) //文本操作对象
		if err != nil {
			return "", err
		}

		ex, err := extractor.New(page)
		if err != nil {
			return "", err
		}

		text, err := ex.ExtractText() //文本
		if err != nil {
			return "", err
		}
		//拼接
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

func extractWord(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	doc, err := document.Read(f, fh.Size) //word文件对象
	if err != nil {
		return "", err
	}

	var allText strings.Builder

	extracted := doc.ExtractText()
	for _, e := range extracted.Items {
		allText.WriteString(e.Text)
	}
	return allText.String(), nil
}

func extractPPT(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	ppt, err := presentation.Read(f, fh.Size)
	if err != nil {
		return "", err
	}
	pt := ppt.ExtractText()
	var allText strings.Builder
	for _, slide := range pt.Slides { //每个  slide  代表一张幻灯片
		for _, item := range slide.Items { //当前这页ppt中的文本项的列表
			allText.WriteString(item.Text)
			allText.WriteString("\n")
		}
	}
	return allText.String(), nil
}
