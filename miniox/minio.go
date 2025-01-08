package miniox

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"strconv"
)

// MinioClient minio客户端
type MinioClient struct {
	c *minio.Client
}

// NewMinioClient 初始化minio客户端
// 	  endpoint url（例如：http://localhost:9000）
//    accessKey 密钥的key
//    secretKey 密钥
//    useSSL    是否使用SSL
func NewMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*MinioClient, error) {
	// 创建minio客户端
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &MinioClient{
		c: minioClient,
	}, nil
}

// Upload 单个文件上传
func (c *MinioClient) Upload(
	ctx context.Context, // 上下文
	bucket string,       // 存储桶
	fileName string,     // 文件名
	contentType string,  // 文件类型，例如：image/png，text/plain，application/json
	data []byte,         // 文件内容
) (minio.UploadInfo, error) {
	// 将对象上传到minio的桶中
	object, err := c.c.PutObject(
		ctx,
		bucket,
		fileName,
		bytes.NewBuffer(data), // 将[]byte转为，io.Reader接口对象（从数据源读取字节流）
		int64(len(data)),      // 文件大小
		minio.PutObjectOptions{ContentType: contentType},
	)
	return object, err
}

// Compose 合并分片
func (c *MinioClient) Compose(
	ctx context.Context, // 上下文
	bucket string,       // 存储桶
	fileName string,     // 文件名
	totalChunk int,      // 分片总数
) error {
	// 目标对象
	dstOpts := minio.CopyDestOptions{
		Bucket: bucket,   // 目标桶名称
		Object: fileName, // 目标对象名字
	}
	var srcs []minio.CopySrcOptions
	// 遍历分片
	for i := 1; i <= totalChunk; i++ {
		// 将分片编号转为字符串
		idx := strconv.Itoa(i)
		// 分片对象
		src := minio.CopySrcOptions{
			Bucket: bucket,
			Object: fileName + "_" + idx,
		}
		// 将分片对象添加到srcs中
		srcs = append(srcs, src)
	}
	// 合并分片，将srcs添加到dstOpts中
	_, err := c.c.ComposeObject(ctx, dstOpts, srcs...)
	return err
}
