package main

import (
	"bytes"
	"io"
	"log"
	"os/exec"
	"tiktok/middleware/oss"
)

func getPicture(r io.Reader) ([]byte, error) {
	var buffer bytes.Buffer
	_, err := io.Copy(&buffer, r)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func screenCaptureAndUpload(title string) bool {
	videoUrl := oss.GetUrl(title, oss.TypeVideo)
	cmd := exec.Command("ffmpeg", "-i", videoUrl, "-frames:v", "1", "-q:v", "2", "-f", "image2pipe", "-vcodec", "mjpeg", "pipe:1")
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start FFmpeg, detail: %v", err)
	}

	pic, err := getPicture(stdout)
	if err != nil {
		log.Printf("failed to read picture(url:%s), detail: %v", videoUrl, err)
		return false
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("occurs a error, detail: %v", err)
	}

	data := bytes.NewBuffer(pic)

	obj := oss.OssObject{
		T:    oss.TypeCover,
		Name: title,
		Data: data,
	}
	oss.StoreObject(obj)
	return true
}
