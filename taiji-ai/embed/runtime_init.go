package main

import (
	"context"
	"errors"

	"github.com/coditary/taiji/internal/runtime"
)

var errRuntimeStarting = errors.New("runtime is still starting")

func (h *runtimeHandle) startRuntimeInit(localMode bool) {
	done := make(chan struct{})
	h.rtInitDone = done
	go func() {
		defer close(done)
		rt, err := runtime.New(context.Background(), h.cfg, runtime.Options{Local: localMode, Quiet: true})
		if rt != nil && rt.Questions != nil {
			rt.Questions.SetPrompter(nil)
		}
		h.rtMu.Lock()
		h.rt = rt
		h.rtInitErr = err
		h.rtReady = true
		h.rtMu.Unlock()
	}()
}

func (h *runtimeHandle) waitRuntimeInit() {
	if h.rtInitDone == nil {
		return
	}
	<-h.rtInitDone
}

func (h *runtimeHandle) runtimeReady() bool {
	h.rtMu.RLock()
	defer h.rtMu.RUnlock()
	return h.rtReady && h.rtInitErr == nil && h.rt != nil
}

func (h *runtimeHandle) runtimeInitError() error {
	h.rtMu.RLock()
	defer h.rtMu.RUnlock()
	if !h.rtReady {
		return errRuntimeStarting
	}
	return h.rtInitErr
}

func (h *runtimeHandle) getRuntime() (*runtime.Runtime, error) {
	h.rtMu.RLock()
	if !h.rtReady {
		h.rtMu.RUnlock()
		return nil, errRuntimeStarting
	}
	err := h.rtInitErr
	rt := h.rt
	h.rtMu.RUnlock()
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return nil, errRuntimeStarting
	}
	return rt, nil
}

func (h *runtimeHandle) closeRuntime() {
	h.waitRuntimeInit()
	h.rtMu.Lock()
	rt := h.rt
	h.rt = nil
	h.rtMu.Unlock()
	if rt != nil {
		_ = rt.Close()
	}
}
