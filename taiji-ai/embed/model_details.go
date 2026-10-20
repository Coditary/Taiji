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

//export taiji_provider_model_details
func taiji_provider_model_details(handle C.uint64_t, providerID *C.char, modelID *C.char) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		return marshalModelDetailsError("runtime not ready")
	}
	details, err := provider.GetProviderModelDetails(context.Background(), h.cfg, C.GoString(providerID), C.GoString(modelID))
	if err != nil {
		return marshalModelDetailsError(err.Error())
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return marshalModelDetailsError("marshal failed")
	}
	return C.CString(string(raw))
}

func marshalModelDetailsError(msg string) *C.char {
	raw, err := json.Marshal(map[string]string{"error": msg})
	if err != nil {
		return C.CString(`{"error":"marshal failed"}`)
	}
	return C.CString(string(raw))
}
