package hachibi

import (
	"context"
	"net/http"
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

	HttpData

	preProcessor  PreProcessor
	processor     Processor
	postProcessor PostProcessor
	errorHandler  ErrorHandler
}

type Processor interface {
	Process(ctx context.Context, httpData *HttpData) error
}

type PreProcessor interface {
	PreProcess(ctx context.Context, httpData *HttpData) error
}

type PostProcessor interface {
	PostProcessor(ctx context.Context, httpData *HttpData) error
}

type ErrorHandler interface {
	ErrorHandle(ctx context.Context, e Error)
}

func NewTransport(opts ...TransportOpt) *Transport {
	t := &Transport{
		originalRoundTripper: http.DefaultTransport,
		HttpData:             HttpData{Error: nil},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

func (t *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	tNow := time.Now().Local()
	ctx := request.Context()

	var response *http.Response

	if err := t.extractRequest(request); err != nil {
		t.HttpData.AppendError(err)
	}

	defer func() {

		if response != nil {
			if err := t.extractResponse(response); err != nil {
				t.HttpData.AppendError(err)
			}
		}

		currentTime := time.Now().Local()
		t.Duration = currentTime.Sub(tNow).Milliseconds()

		if t.preProcessor != nil {
			if err := t.preProcessor.PreProcess(ctx, &t.HttpData); err != nil {
				err = errors.Wrap(err, "pre process error")
				t.HttpData.AppendError(err)
			}
		}

		if t.processor != nil {
			if err := t.processor.Process(ctx, &t.HttpData); err != nil {
				err = errors.Wrap(err, "process error")
				t.HttpData.AppendError(err)
			}
		}

		if t.postProcessor != nil {
			if err := t.postProcessor.PostProcessor(ctx, &t.HttpData); err != nil {
				err = errors.Wrap(err, "post process error")
				t.HttpData.AppendError(err)
			}
		}

		if t.Error != nil && t.errorHandler != nil {
			t.errorHandler.ErrorHandle(ctx, *t.Error)
		}
	}()

	response, errRoundTrip := t.originalRoundTripper.RoundTrip(request)
	if errRoundTrip != nil {
		t.HttpData.AppendError(errRoundTrip)
		return nil, errRoundTrip
	}

	return response, nil
}
