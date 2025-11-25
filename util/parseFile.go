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
	"sync"

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
	mimeTypeZip  = "application/zip"
)

// 文件解析器接口
type FileParser interface {
	// Supports 检查是否支持解析该文件类型
	Supports(mimeType, ext string) bool
	// Parse 解析文件内容
	Parse(fh *multipart.FileHeader) (string, error)
	// Name 返回解析器名称
	Name() string
}

// 解析器注册表
type ParserRegistry struct {
	parsers []FileParser
	mu      sync.RWMutex
}

var (
	globalRegistry = &ParserRegistry{}
	once           sync.Once
)

// 初始化并注册所有解析器
func initRegistry() {
	globalRegistry.Register(&PDFParser{})
	globalRegistry.Register(&WordParser{})
	globalRegistry.Register(&PPTParser{})
}

// GetRegistry 获取全局解析器注册表
func GetRegistry() *ParserRegistry {
	once.Do(initRegistry)
	return globalRegistry
}

// Register 注册文件解析器
func (r *ParserRegistry) Register(parser FileParser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers = append(r.parsers, parser)
}

// GetParser 根据MIME类型和扩展名获取合适的解析器
func (r *ParserRegistry) GetParser(mimeType, ext string) FileParser {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, parser := range r.parsers {
		if parser.Supports(mimeType, ext) {
			return parser
		}
	}
	return nil
}

// PDF解析器
type PDFParser struct{}

func (p *PDFParser) Supports(mimeType, ext string) bool {
	return mimeType == mimeTypePDF || ext == ".pdf"
}

func (p *PDFParser) Parse(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	pdfReader, err := model.NewPdfReader(f)
	if err != nil {
		return "", err
	}

	if pdfReader == nil {
		return "", fmt.Errorf("暂不支持解析该文件")
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", err
	}

	var textBuilder strings.Builder

	for i := 0; i < numPages; i++ {
		pageNum := i + 1

		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			return "", err
		}

		ex, err := extractor.New(page)
		if err != nil {
			return "", err
		}

		text, err := ex.ExtractText()
		if err != nil {
			return "", err
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

func (p *PDFParser) Name() string {
	return "PDFParser"
}

// Word文档解析器（支持.doc和.docx）
type WordParser struct{}

func (p *WordParser) Supports(mimeType, ext string) bool {
	return mimeType == mimeTypeDoc || mimeType == mimeTypeDocx ||
		mimeType == mimeTypeZip && ext == ".docx" ||
		ext == ".doc" || ext == ".docx"
}

func (p *WordParser) Parse(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	doc, err := document.Read(f, fh.Size)
	if err != nil {

		return "", fmt.Errorf("文档读取失败（可能格式不支持或文件损坏）：%w", err)
	}
	if doc == nil {
		return "", fmt.Errorf("文档为空，暂不支持解析")
	}

	var allText strings.Builder
	extracted := doc.ExtractText()

	if extracted == nil {
		return "", fmt.Errorf("暂不支持解析该文件")
	}
	for _, e := range extracted.Items {
		allText.WriteString(e.Text)
	}
	return allText.String(), nil
}

func (p *WordParser) Name() string {
	return "WordParser"
}

// PPT解析器（支持.ppt和.pptx）
type PPTParser struct{}

func (p *PPTParser) Supports(mimeType, ext string) bool {
	return mimeType == mimeTypePPT || mimeType == mimeTypePPTx ||
		mimeType == mimeTypeZip && ext == ".pptx" ||
		ext == ".ppt" || ext == ".pptx"
}

func (p *PPTParser) Parse(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	ppt, err := presentation.Read(f, fh.Size)
	if err != nil {
		return "", err
	}
	if ppt == nil {
		return "", fmt.Errorf("暂不支持解析该文件")
	}

	pt := ppt.ExtractText()
	var allText strings.Builder
	for _, slide := range pt.Slides {
		for _, item := range slide.Items {
			allText.WriteString(item.Text)
			allText.WriteString("\n")
		}
	}
	return allText.String(), nil
}

func (p *PPTParser) Name() string {
	return "PPTParser"
}

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

	// 使用注册表获取合适的解析器
	parser := GetRegistry().GetParser(mime, ext)
	if parser == nil {
		zlog.CtxErrorf(ctx, "no parser found for file %s: MIME=%s, ext=%s", fh.Filename, mime, ext)
		return "", fmt.Errorf("unsupported file type: MIME=%s, ext=%s", mime, ext)
	}

	zlog.CtxInfof(ctx, "using parser %s for file %s", parser.Name(), fh.Filename)
	text, err = parser.Parse(fh)

	if err != nil {
		zlog.CtxErrorf(ctx, "failed to extract content from %s using %s: %v", fh.Filename, parser.Name(), err)
		return "", err
	}

	return text, nil
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
