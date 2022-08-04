package hachibi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Payload struct {
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

type Request struct {
	Payload
}

type Response struct {
	Payload
}

type Transport struct {
	originalRoundTripper http.RoundTripper

	Request  Request  `json:"request"`
	Response Response `json:"response"`

	Duration   int64  `json:"duration"`
	URL        string `json:"url"`
	Method     string `json:"method"`
	StatusCode int    `json:"statusCode"`

	Event string `json:"event"`

	processor Processor
}

type Processor interface {
	Process(transport Transport) error
}

type TransportOpt func(*Transport)

func WithProcessingData(processor Processor) TransportOpt {
	return func(transport *Transport) {
		transport.processor = processor
	}
}

func WithEventName(name string) TransportOpt {
	return func(transport *Transport) {
		transport.Event = name
	}
}

func NewTransport(opts ...TransportOpt) *Transport {
	t := &Transport{
		originalRoundTripper: http.DefaultTransport,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

func (t *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	tNow := time.Now().Local()

	if err := t.extractRequest(request); err != nil {
		return nil, errors.Wrap(err, "[hachibi]: extract request")
	}

	response, err := t.originalRoundTripper.RoundTrip(request)
	if err != nil {
		return nil, err
	}

	if response != nil {
		if err := t.extractResponse(response); err != nil {
			return nil, errors.Wrap(err, "[hachibi]: extract response")
		}
	}

	currentTime := time.Now().Local()
	t.Duration = currentTime.Sub(tNow).Milliseconds()

	if t.processor != nil {
		if err := t.processor.Process(*t); err != nil {
			return nil, errors.Wrap(err, "[hachibi]: process request and response")
		}
	}

	return response, nil
}

func (t *Transport) extractResponse(r *http.Response) error {
	t.StatusCode = r.StatusCode
	t.Response.Header = r.Header

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body.Close()

	t.Response.Body = bodyBytes

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return nil
}

func (t *Transport) extractRequest(r *http.Request) error {
	t.URL = r.URL.String()
	t.Method = r.Method
	t.Request.Header = r.Header

	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		return t.exportMultipartFormData(r)
	}

	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return err
		}

		t.Request.Body = body

		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	return nil
}

type fileData struct {
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
	File     []byte `json:"file"`
}

func (t *Transport) exportMultipartFormData(request *http.Request) error {
	r := request.Clone(request.Context())
	// copy the body
	rBody, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	request.Body.Close()

	//fill body
	request.Body = io.NopCloser(bytes.NewBuffer(rBody))
	r.Body = io.NopCloser(bytes.NewBuffer(rBody))

	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		return err
	}

	m := r.MultipartForm

	body := map[string]any{}

	files := m.File
	for key, fs := range files {
		var fileBody any

		if len(fs) > 1 {

			allFiles := make([]fileData, 0)

			for _, f := range fs {
				file, err := extractFileData(f)
				if err != nil {
					return err
				}

				allFiles = append(allFiles, *file)

			}

			fileBody = allFiles

		} else {
			file, err := extractFileData(fs[0])
			if err != nil {
				return err
			}

			fileBody = file
		}

		body[key] = fileBody
	}

	for key, v := range m.Value {
		if len(v) > 1 {
			body[key] = v
			continue
		}

		body[key] = v[0]
	}

	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	t.Request.Body = b

	return nil
}

func extractFileData(f *multipart.FileHeader) (*fileData, error) {
	file, err := f.Open()
	if err != nil {
		return nil, err
	}

	fileByte, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return &fileData{
		FileName: f.Filename,
		Size:     f.Size,
		File:     fileByte,
	}, nil
}
