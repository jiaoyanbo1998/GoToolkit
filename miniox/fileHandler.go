package miniox

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"mime/multipart"
	"strconv"
	"sync"
	"time"
)

const (
	defaultChunkSize = 100 * 1024 * 1024 // 默认的分片大小：50MB
	maxConcurrency   = 5                 // 最大并发上传数
)

type FileHandler struct {
	minio *MinioClient
}

func NewFileHandler(minio *MinioClient) *FileHandler {
	return &FileHandler{
		minio: minio,
	}
}

// CalculateTotalChunks 根据文件大小计算分片数量
func (f *FileHandler) calculateTotalChunks(fileSize int64) int {
	if fileSize <= 0 {
		return 1 // 如果文件大小为0或负数，按单文件处理
	}
	totalChunks := fileSize / defaultChunkSize
	if fileSize%defaultChunkSize != 0 {
		totalChunks++
	}
	return int(totalChunks)
}

// UploadResult 用于返回上传结果
type UploadResult struct {
	FileName string
	URL      string
	Error    error
}

// contentTypeToExt 文件类型 ==> 文件扩展名
var contentTypeToExt = map[string]string{
	"image/jpeg":       ".jpg",
	"image/png":        ".png",
	"application/pdf":  ".pdf",
	"text/plain":       ".txt",
	"application/zip":  ".zip",
	"application/json": ".json",
	"application/xml":  ".xml",
	"video/mp4":        ".mp4",
}

func (f *FileHandler) UploadFiles(ctx *gin.Context, endpoint, bucketName, fileName string,
	useSSL bool, timeout time.Duration) ([]UploadResult, error) {
	// 获取文件信息
	multipartForm, err := ctx.MultipartForm()
	if err != nil {
		return nil, errors.New("获取文件信息失败")
	}
	// 获取文件列表
	files := multipartForm.File["file"]
	// 判断文件列表是否为空
	if len(files) == 0 {
		return nil, errors.New("文件列表为空")
	}
	// 使用 WaitGroup 控制并发
	var wg sync.WaitGroup
	resultChan := make(chan UploadResult, len(files))
	// 限制并发数
	concurrencyLimit := make(chan struct{}, maxConcurrency)
	for _, file := range files {
		// 添加1个等待
		wg.Add(1)
		// 占用一个并发槽
		concurrencyLimit <- struct{}{}
		// 启动一个goroutine处理每个文件上传
		go func(file *multipart.FileHeader) {
			// 完成后减少1个等待
			defer wg.Done()
			// 释放并发槽
			defer func() {
				<-concurrencyLimit
			}()
			var url string
			var er error
			// 计算分片数量
			totalChunks := f.calculateTotalChunks(file.Size)
			if totalChunks == 1 {
				// 不分片上传
				url, er = f.onlyPickUpload(file, bucketName, fileName, endpoint, useSSL, timeout)
			} else {
				// 分片上传
				// url, er = f.manyPickUpload(file, bucketName, fileName, endpoint, totalChunks, useSSL, timeout)
				url, er = f.manyPickSyncUpload(file, bucketName, fileName, endpoint, totalChunks, useSSL, timeout)
			}
			// 将结果发送到channel
			resultChan <- UploadResult{
				FileName: file.Filename,
				URL:      url,
				Error:    er,
			}
		}(file)
	}
	// 等待所有文件上传完成
	wg.Wait()
	close(resultChan) // 关闭channel
	// 收集结果
	results := make([]UploadResult, 0, len(files))
	for result := range resultChan {
		results = append(results, result)
	}
	return results, nil
}

// ManyPickUpload 分片上传
func (f *FileHandler) manyPickUpload(file *multipart.FileHeader, bucketName, fileName,
	endpoint string, totalChunks int, useSSL bool, timeout time.Duration) (string, error) {
	// 文件大小
	sizeSum := int(file.Size)
	// 文件在minio中的路径
	filePath := ""
	// 文件类型
	contentType := file.Header.Get("Content-Type")
	// 文件扩展名
	ext, ok := contentTypeToExt[contentType]
	if !ok {
		// 没有找到对应的扩展名，使用默认扩展名
		ext = ".bin"
	}
	// 打开文件
	open, er := file.Open()
	if er != nil {
		return "", errors.New("打开文件失败")
	}
	// 关闭文件
	defer open.Close()
	// 分片大小
	size := defaultChunkSize
	for i := 0; i < totalChunks; i++ {
		// 计算最后一个分片的大小
		if i == totalChunks-1 {
			size = sizeSum - i*size
		}
		buf := make([]byte, size)
		// 读取文件内容
		n, err := open.Read(buf)
		if err != nil {
			return "", errors.New("读取文件内容失败")
		}
		// 编号
		idx := strconv.Itoa(i)
		// 上传文件
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		fName := fmt.Sprintf("%s_%s%s", fileName, idx, ext)
		_, er = f.minio.Upload(ctx, bucketName, fName, contentType, buf[:n])
		if er != nil {
			return "", errors.New("上传分片失败")
		}
		cancel()
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), timeout)
	defer cancel2()
	// 合并文件
	err := f.minio.Compose(ctx2, bucketName, fileName, totalChunks, ext)
	if err != nil {
		return "", errors.New("文件合并失败")
	}
	// 文件路径
	filePath = endpoint + "/" + bucketName + "/" + fileName + ext
	if useSSL {
		filePath = "https://" + filePath
	} else {
		filePath = "http://" + filePath
	}
	return filePath, nil
}

func (f *FileHandler) onlyPickUpload(file *multipart.FileHeader, bucketName, fileName,
	endpoint string, useSSL bool, timeout time.Duration) (string, error) {
	// 打开文件
	open, er := file.Open()
	if er != nil {
		return "", errors.New("打开文件失败")
	}
	defer open.Close()
	// 读取文件内容
	buf := make([]byte, file.Size)
	n, er := open.Read(buf)
	if er != nil {
		return "", errors.New("读取文件内容失败")
	}
	// 文件类型
	contentType := file.Header.Get("Content-Type")
	// 文件扩展名
	ext, ok := contentTypeToExt[contentType]
	if !ok {
		// 没有找到对应的扩展名，使用默认扩展名
		ext = ".bin"
	}
	// 拼接后的文件名
	fileName = fmt.Sprintf("%s%s", fileName, ext)
	// 上传文件
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	_, er = f.minio.Upload(ctx, bucketName, fileName, contentType, buf[:n])
	if er != nil {
		return "", errors.New("上传文件失败")
	}
	// 返回文件路径
	filePath := endpoint + "/" + bucketName + "/" + fileName
	if useSSL {
		filePath = "https://" + filePath
	} else {
		filePath = "http://" + filePath
	}
	cancel()
	return filePath, nil
}

// DeleteFile 删除文件
func (f *FileHandler) DeleteFile(bucketName, fileName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// 删除文件
	er := f.minio.Delete(ctx, bucketName, fileName)
	if er != nil {
		return errors.New("删除文件失败")
	}
	return nil
}

// CheckFileExist 检查文件是否存在
func (f *FileHandler) CheckFileExist(bucketName, fileName string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// 检查文件是否存在
	exist, er := f.minio.CheckFileExists(ctx, bucketName, fileName)
	// 文件不存在
	if er != nil {
		return false
	}
	// 文件存在
	return exist
}

// 分片异步上传
func (f *FileHandler) manyPickSyncUpload(file *multipart.FileHeader, bucketName, fileName,
	endpoint string, totalChunks int, useSSL bool, timeout time.Duration) (string, error) {
	// 文件大小
	sizeSum := int(file.Size)
	// 文件在minio中的路径
	filePath := ""
	// 文件类型
	contentType := file.Header.Get("Content-Type")
	// 文件扩展名
	ext, ok := contentTypeToExt[contentType]
	if !ok {
		// 没有找到对应的扩展名，使用默认扩展名
		ext = ".bin"
	}
	// 打开文件
	open, er := file.Open()
	if er != nil {
		return "", errors.New("打开文件失败")
	}
	// 关闭文件
	defer open.Close()

	// 使用 WaitGroup 来等待所有协程完成
	var wg sync.WaitGroup
	// 用于收集错误
	errChan := make(chan error, totalChunks)

	// 分片大小
	size := defaultChunkSize
	for i := 0; i < totalChunks; i++ {
		// 计算最后一个分片的大小
		if i == totalChunks-1 {
			size = sizeSum - i*size
		}
		buf := make([]byte, size)
		// 读取文件内容
		n, err := open.Read(buf)
		if err != nil {
			return "", fmt.Errorf("读取文件内容失败: %w", err)
		}

		// 启动一个协程来处理分片上传
		wg.Add(1)
		go func(i int, buf []byte, n int) {
			// 完成后减少1个等待
			defer wg.Done()

			// 编号
			idx := strconv.Itoa(i)
			// 上传文件
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			fName := fmt.Sprintf("%s_%s%s", fileName, idx, ext)
			_, err := f.minio.Upload(ctx, bucketName, fName, contentType, buf[:n])
			if err != nil {
				errChan <- fmt.Errorf("上传分片 %d 失败: %v", i, err)
				return
			}
			cancel()
		}(i, buf, n)
	}

	// 等待所有协程完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误发生
	for err := range errChan {
		if err != nil {
			return "", err
		}
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), timeout)
	defer cancel2()
	// 合并文件
	err := f.minio.Compose(ctx2, bucketName, fileName, totalChunks, ext)
	if err != nil {
		return "", fmt.Errorf("文件合并失败: %w", err)
	}
	// 文件路径
	filePath = endpoint + "/" + bucketName + "/" + fileName + ext
	if useSSL {
		filePath = "https://" + filePath
	} else {
		filePath = "http://" + filePath
	}
	return filePath, nil
}
