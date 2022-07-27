package aws_s3

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/CPU-commits/Intranet_BClassroom/settings"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
)

type AWSS3 struct {
	sess *session.Session
}

var settingsData = settings.GetSettings()

func NewAWSS3() *AWSS3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(settingsData.AWS_REGION),
	}))
	return &AWSS3{
		sess: sess,
	}
}

func (aws_s3 *AWSS3) DeleteFile(key string) error {
	svc := s3.New(aws_s3.sess)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(settingsData.AWS_BUCKET),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(settingsData.AWS_BUCKET),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	return nil
}

func (aws_s3 *AWSS3) UploadFile(file *multipart.FileHeader) (*s3manager.UploadOutput, error) {
	ext := strings.Split(file.Filename, ".")
	uploader := s3manager.NewUploader(aws_s3.sess)
	// To buffer
	openFile, err := file.Open()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, openFile); err != nil {
		return nil, err
	}
	fileName := uuid.New()
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(settingsData.AWS_BUCKET),
		Key:    aws.String(fmt.Sprintf("classroom/%s.%s", fileName.String(), ext[len(ext)-1])),
		Body:   buf,
	})
	return result, err
}

func (aws_s3 *AWSS3) GetFile(key string) (io.ReadCloser, error) {
	svc := s3.New(aws_s3.sess)
	file, err := svc.GetObject(&s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(settingsData.AWS_BUCKET),
	})
	if err != nil {
		return nil, err
	}
	return file.Body, nil
}
