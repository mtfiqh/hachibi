package hachibi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type HttpData struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`

	Duration   int64  `json:"duration"`
	URL        string `json:"url"`
	Method     string `json:"method"`
	StatusCode int    `json:"statusCode"`

	Event string `json:"event"`

	Error Error `json:"error"`
}

func (h *HttpData) AppendError(e error) {
	if h.Error == nil {
		h.Error = make(Error, 0)
	}

	h.Error = append(h.Error, e)
}

func (t *HttpData) extractResponse(r *http.Response) error {
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

func (t *HttpData) extractRequest(r *http.Request) error {
	t.URL = r.URL.String()
	t.Method = r.Method
	t.Request.Header = r.Header.Clone()

	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		return t.exportMultipartFormData(r)
	}

	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return errors.Wrap(err, "failed to read all request body")
		}

		t.Request.Body = body

		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	return nil
}

type MultipartFileData struct {
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
	File     []byte `json:"file"`
}

func (t *HttpData) exportMultipartFormData(request *http.Request) error {
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

			allFiles := make([]MultipartFileData, 0)

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

func extractFileData(f *multipart.FileHeader) (*MultipartFileData, error) {
	file, err := f.Open()
	if err != nil {
		return nil, err
	}

	fileByte, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return &MultipartFileData{
		FileName: f.Filename,
		Size:     f.Size,
		File:     fileByte,
	}, nil
}

func (t HttpData) GetMultipartFileDataFromRequest(key string) ([]MultipartFileData, error) {
	if !strings.Contains(t.Request.Header.Get("Content-Type"), "multipart/form-data") {
		return nil, errors.New("not multipart form data")
	}

	body := make(map[string]any)
	if err := json.Unmarshal(t.Request.Body, &body); err != nil {
		return nil, errors.Wrap(err, "failed unmarshal to body")
	}

	fileDataBytes, err := json.Marshal(body[key])
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal body")
	}

	fileDatas := make([]MultipartFileData, 0)
	fileData := MultipartFileData{}

	err = json.Unmarshal(fileDataBytes, &fileData)
	if err != nil {
		err := json.Unmarshal(fileDataBytes, &fileDatas)
		if err != nil {
			return nil, errors.Wrap(err, "failed unmarshal to slice of multipart file data")
		}
	} else {
		fileDatas = append(fileDatas, fileData)
	}

	return fileDatas, nil
}
