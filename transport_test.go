package hachibi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func (p Processor) Process(transport hachibi.Transport) error {

	var requestBody any
	err := json.Unmarshal(transport.Response.Body, &requestBody)
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
		err := p.Transformer.Transform(&transport)
		if err != nil {
			return err
		}
	}

	if p.db != nil {
		query := `insert into logs (id, request, response, method, url, duration, status_code, created_at) values (:id, :request, :response, :method, :url, :duration, :status_code, :created_at)`

		requestBytes, _ := json.Marshal(request)
		responseBytes, err := json.Marshal(response)
		req := json.RawMessage(requestBytes)
		if err != nil {
			return err
		}
		_, err = p.db.NamedExec(query, map[string]any{
			"id":          uuid.New().String(),
			"request":     req,
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

func (t Req) Transform(transport *hachibi.Transport) error {
	if err := json.Unmarshal(transport.Request.Body, &t); err != nil {
		return err
	}

	t["image"] = fmt.Sprintf("%s/%s", "upload", "image-123")

	newBody, err := json.Marshal(t)
	if err != nil {
		return err
	}

	transport.Request.Body = newBody
	return nil
}

func TestTransport_RoundTrip(t *testing.T) {
	transport := hachibi.NewTransport(hachibi.WithProcessingData(NewProcessor()))
	client := http.Client{Transport: transport}
	url := "http://demo9323592.mockable.io/post"

	db, err := sqlx.Open("postgres", "user=postgres dbname=hachibi password=secret port=15432 sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("with application/json post with transformer", func(t *testing.T) {
		transport := hachibi.NewTransport(hachibi.WithProcessingData(NewProcessor(WithDB(db))))
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
		req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
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
