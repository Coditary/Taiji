package main

import (
	"context"
	"errors"
	"testing"
)

func TestIsUserCancellation(t *testing.T) {
	if !isUserCancellation(context.Canceled) {
		t.Fatal("context.Canceled should count as user cancellation")
	}
	grpcErr := errors.New(`remote generate text stream: rpc error: code = Canceled desc = grpc: the client connection is closing`)
	if isUserCancellation(grpcErr) {
		t.Fatal("gRPC transport cancel should not count as user cancellation")
	}
	if !isUserCancellation(errors.New("request cancelled by user")) {
		t.Fatal("plain cancel message should count as user cancellation")
	}
}
