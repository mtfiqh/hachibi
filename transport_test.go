package hachibi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mtfiqh/hachibi"
)

type Transformer interface {
	Transform(transport *hachibi.Transport) error
}

type Processor struct {
	Transformer
	db *sqlx.DB
}

func (p Processor) Process(ctx context.Context, transport *hachibi.Transport) error {

	var requestBody any
	err := json.Unmarshal(transport.Request.Body, &requestBody)
	if err != nil {
		requestBody = transport.Request.Body
	}

	var responseBody any
	err = json.Unmarshal(transport.Response.Body, &responseBody)
	if err != nil {
		responseBody = transport.Response.Body
	}

	request := map[string]any{
		"header": transport.Request.Header,
		"body":   requestBody,
	}

	response := map[string]any{
		"header": transport.Response.Header,
		"body":   responseBody,
	}

	if p.Transformer != nil {
		err := p.Transformer.Transform(transport)
		if err != nil {
			return err
		}
	}
	errB, _ := json.Marshal(transport.Error)
	log.Println(string(errB))
	if p.db != nil {
		query := `insert into logs (id, request, response, method, url, duration, status_code, created_at) values (:id, :request, :response, :method, :url, :duration, :status_code, :created_at)`

		requestBytes, _ := json.Marshal(request)
		responseBytes, err := json.Marshal(response)
		if err != nil {
			return err
		}
		_, err = p.db.NamedExec(query, map[string]any{
			"id":          uuid.New().String(),
			"request":     requestBytes,
			"response":    responseBytes,
			"method":      transport.Method,
			"url":         transport.URL,
			"duration":    transport.Duration,
			"status_code": transport.StatusCode,
			"created_at":  time.Now(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (p Processor) indent(b []byte) []byte {
	var a any
	json.Unmarshal(b, &a)
	bb, _ := json.MarshalIndent(a, "", "   ")
	return bb
}

type ProcessorOpt func(*Processor)

func WithTransformer(transformer Transformer) ProcessorOpt {
	return func(processor *Processor) {
		processor.Transformer = transformer
	}
}

func WithDB(db *sqlx.DB) ProcessorOpt {
	return func(processor *Processor) {
		processor.db = db
	}
}

func NewProcessor(opts ...ProcessorOpt) *Processor {
	p := &Processor{}
	for _, opt := range opts {
		opt(p)
	}

	return p
}

type Request struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Image string `json:"image"`
}

type Req map[string]any

func (t Request) Transform(transport *hachibi.Transport) error {
	if err := json.Unmarshal(transport.Request.Body, &t); err != nil {
		return err
	}

	t.Image = fmt.Sprintf("%s/%s", "upload", "image-123")

	newBody, err := json.Marshal(t)
	if err != nil {
		return err
	}

	transport.Request.Body = newBody
	return nil
}

func (t Req) PreProcess(ctx context.Context, transport *hachibi.Transport) error {
	files, err := transport.GetMultipartFileDataFromRequest("file")
	if err != nil {
		return err
	}

	log.Println(files[0].File)
	return nil
}

func (p Processor) PreProcess(ctx context.Context, transport *hachibi.Transport) error {
	t := Request{}
	if err := json.Unmarshal(transport.Request.Body, &t); err != nil {
		return err
	}

	t.Image = fmt.Sprintf("%s/%s", "upload", "image-123")

	newBody, err := json.Marshal(t)
	if err != nil {
		return err
	}

	transport.Request.Body = newBody
	return nil
}

func (p Processor) ErrorHandle(ctx context.Context, e hachibi.Error) {
	log.Println("error => ", e.Error())
}

func TestTransport_RoundTrip(t *testing.T) {
	db, err := sqlx.Open("postgres", "user=postgres dbname=hachibi password=secret port=15432 sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	tt := Req{}
	pp := NewProcessor(WithDB(db))
	transport := hachibi.NewTransport(hachibi.WithProcessingData(pp), hachibi.WithPreProcessor(tt), hachibi.WithErrorHandle(pp))
	client := http.Client{Transport: transport}
	url := "http://localhost:1234/post"

	t.Run("with application/json post with transformer", func(t *testing.T) {
		//transport := hachibi.NewTransport(hachibi.WithProcessingData(NewProcessor(WithDB(db))))
		client := http.Client{Transport: transport}

		body := struct {
			Email string `json:"email"`
			Phone string `json:"phone"`
			Image string `json:"image"`
		}{
			Email: "mtfiqh@gmail.com",
			Phone: "+6285161101060",
			Image: GetImageTest(),
		}

		b, _ := json.Marshal(body)
		ctx, canc := context.WithTimeout(context.Background(), 6*time.Second)
		defer canc()
		_ = b
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(b))
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(res.StatusCode)
		resB, _ := io.ReadAll(res.Body)
		res.Body.Close()

		t.Log(string(resB))
	})

	t.Run("with application/json post", func(t *testing.T) {
		body := struct {
			Email string `json:"email"`
			Phone string `json:"phone"`
			Image string `json:"image"`
		}{
			Email: "mtfiqh@gmail.com",
			Phone: "+6285161101060",
			Image: GetImageTest(),
		}

		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(res.StatusCode)
		resB, _ := io.ReadAll(res.Body)
		res.Body.Close()

		t.Log(string(resB))
	})

	t.Run("with multipart form data", func(t *testing.T) {
		fileDir, _ := os.Getwd()
		fileName := "test.png"
		filePath := path.Join(fileDir, fileName)

		file, _ := os.Open(filePath)
		defer file.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
		io.Copy(part, file)
		name, _ := writer.CreateFormField("name")
		name.Write([]byte("namaku taufiq"))
		writer.Close()

		r, _ := http.NewRequest("POST", url, body)
		r.Header.Add("Content-Type", writer.FormDataContentType())
		res, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(res.StatusCode)
		resB, _ := io.ReadAll(res.Body)
		res.Body.Close()

		t.Log(string(resB))

	})
}
