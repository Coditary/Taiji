package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"

	"github.com/coditary/taiji/internal/provider"
)

type providerModelsPayload struct {
	ProviderID string                      `json:"provider_id"`
	Models     []provider.ProviderModelOption `json:"models"`
	Error      string                      `json:"error,omitempty"`
}

//export taiji_provider_models
func taiji_provider_models(handle C.uint64_t, providerID *C.char) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	out := providerModelsPayload{}
	if !ok || h == nil || h.cfg == nil {
		out.Error = "runtime not ready"
		return marshalProviderModels(out)
	}
	id := C.GoString(providerID)
	out.ProviderID = id
	models, err := provider.ListProviderModels(context.Background(), h.cfg, id)
	if err != nil {
		out.Error = err.Error()
		return marshalProviderModels(out)
	}
	out.Models = models
	return marshalProviderModels(out)
}

func marshalProviderModels(payload providerModelsPayload) *C.char {
	raw, err := json.Marshal(payload)
	if err != nil {
		return C.CString(`{"error":"marshal failed"}`)
	}
	return C.CString(string(raw))
}
