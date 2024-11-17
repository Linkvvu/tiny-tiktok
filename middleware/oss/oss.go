package oss

import (
	"fmt"
	"io"
	"log"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type ObjType int

const (
	TypeVideo ObjType = iota
	TypeCover
)

func getTypeString(t ObjType) string {
	switch t {
	case TypeVideo:
		return "video"
	case TypeCover:
		return "cover"
	default:
		panic("invalid ObjType")
	}
}

var ossBucket *oss.Bucket
var bucketName = "proj-tiktok"
var endpoint = "oss-cn-huhehaote.aliyuncs.com"

// var ossChan chan OssObject
// var defaultChanSize = 6

type OssObject struct {
	T    ObjType
	Name string
	Data io.Reader
}

func (o OssObject) GetKey() string {
	var suffix string
	switch o.T {
	case TypeVideo:
		suffix = ".mp4"
	case TypeCover:
		suffix = ".jpg"
	}
	return fmt.Sprintf("%s/%s%s", getTypeString(o.T), o.Name, suffix)
}

func init() {
	var err error
	var ossClient *oss.Client
	provider, _ := oss.NewEnvironmentVariableCredentialsProvider()
	cred := provider.GetCredentials()
	ossClient, err = oss.New(endpoint, cred.GetAccessKeyID(), cred.GetAccessKeySecret())
	if err != nil {
		log.Panicln("Failed to connect OSS")
	}
	ossBucket, err = ossClient.Bucket(bucketName)
	if err != nil {
		log.Panicln("Failed to get target bucket from OSS")
	}
	// ossChan = make(chan OssObject, defaultChanSize)

	// go handleOss()
}

func StoreObject(obj OssObject) error {
	return ossBucket.PutObject(obj.GetKey(), obj.Data)
}

// func handleOss() {
// 	for obj := range ossChan {
// 		err := ossBucket.PutObject(obj.GetKey(), obj.Data)
// 		// log and discard
// 		if err != nil {
// 			log.Println(err.Error())
// 		}
// 	}
// }

func GetUrl(base string, t ObjType) string {
	switch t {
	case TypeVideo:
		base += ".mp4"
	case TypeCover:
		base += ".jpg"
	}
	return fmt.Sprintf("https://%s.%s/%s/%s", bucketName, endpoint, getTypeString(t), base)
}
