package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/coditary/taiji/internal/runtime"
	grpcserver "github.com/coditary/taiji/internal/server/grpc"
)

func newServeCmd(app *App) *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Taiji gRPC server",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := runtime.New(cmd.Context(), app.Config, app.runtimeOptions(cmd))
			if err != nil {
				return err
			}
			defer rt.Close()
			srv := grpcserver.New(rt)
			fmt.Fprintf(cmd.ErrOrStderr(), "taiji gRPC listening on %s\n", addr)
			return srv.Serve(addr)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:50052", "listen address")
	return cmd
}
