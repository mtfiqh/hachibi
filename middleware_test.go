package hachibi_test

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtfiqh/hachibi"
)

type P struct {
}

func (p P) Process(ctx context.Context, httpData *hachibi.HttpData) error {
	log.Println(httpData.Request.Header)
	return nil
}

func (p P) PreProcess(ctx context.Context, httpData *hachibi.HttpData) error {
	httpData.Request.Header.Set("Authorization", "basic sensor")
	return nil
}

func TestMiddleware(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "basic hahahahaa")
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		writer.Header().Set("Authorization", "basic abcd")
		data := struct {
			Name string `json:"name"`
			DOB  string `json:"dob"`
		}{
			Name: "aku namanya",
			DOB:  "dob ehehe",
		}

		db, _ := json.Marshal(data)

		writer.Write(db)
	})

	m := hachibi.NewMiddleware(hachibi.MiddlewareWithProcessor(P{}))

	t.Run("event name satu", func(t *testing.T) {
		h := m.Middleware(m.PreProcessMiddleware(P{})(m.SetEventName("satu")(handler)))

		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)

		t.Log("res =>", res.Result())
		b, _ := io.ReadAll(res.Body)
		t.Log("body => ", string(b))

	})

	t.Run("event name satu", func(t *testing.T) {
		h := m.Middleware(m.SetEventName("dua")(handler))

		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)

		t.Log("res =>", res.Result())
		b, _ := io.ReadAll(res.Body)
		t.Log("body => ", string(b))
	})

}
