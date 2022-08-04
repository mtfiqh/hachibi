package example

import (
	"encoding/json"
	"fmt"
	"hachibi"
	"net/http"
)

type LoggerAdditional interface {
	ProcessingImage(transport *hachibi.Transport) error
}

type Logger struct {
	//	 repo

	LoggerAdditional
}

type LoggerImage struct {
	//	repo gcs
	Request struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
		Image string `json:"image"`
	}
}

func (l LoggerImage) ProcessingImage(transport *hachibi.Transport) error {
	request := LoggerImage{}

	json.Unmarshal(transport.Request.Body, &request)

	imageB64 := request.Request.Image

	imageB64 = fmt.Sprintf("upload/image-123")

	request.Request.Image = imageB64

	requestB, _ := json.Marshal(request.Request)

	transport.Request.Body = requestB

	return nil
}

func (l Logger) Process(transport hachibi.Transport) error {
	//	todo inser db
	if l.LoggerAdditional != nil {
		l.LoggerAdditional.ProcessingImage(&transport)
	}
	return nil
}

func (l Logger) AdditionalData(additional LoggerAdditional) Logger {
	logger := l
	logger.LoggerAdditional = additional
	return logger
}

func DoLiveness() {

	transport := hachibi.NewTransport(hachibi.WithProcessingData(Logger{}.AdditionalData(LoggerImage{})))
	client := http.Client{Transport: transport}

	req, _ := http.NewRequest("", "", nil)

	res, _ := client.Do(req)

	_ = res
}
