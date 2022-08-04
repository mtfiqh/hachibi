package hachibi

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

func WithErrorHandle(handler ErrorHandler) TransportOpt {
	return func(transport *Transport) {
		transport.ErrorHandler = handler
	}
}

func WithPreProcessor(p PreProcessor) TransportOpt {
	return func(transport *Transport) {
		transport.preProcessor = p
	}
}

func WithPostProcessor(p PostProcessor) TransportOpt {
	return func(transport *Transport) {
		transport.postProcessor = p
	}
}
