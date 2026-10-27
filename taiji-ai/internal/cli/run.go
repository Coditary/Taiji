package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/coditary/taiji/internal/agent"
	"github.com/coditary/taiji/internal/agentsdoc"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
)

func newRunCmd(app *App) *cobra.Command {
	var providerID string
	var skillNames []string
	var sessionID string

	cmd := &cobra.Command{
		Use:   "run <goal>",
		Short: "Run an agent loop for a goal",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			goal := strings.Join(args, " ")
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			rt, err := runtime.New(ctx, app.Config, app.runtimeOptions(cmd))
			if err != nil {
				return err
			}
			defer rt.Close()

			var sess *session.Session
			if sessionID != "" {
				sess, err = session.Load(app.Config.Root, sessionID)
				if err != nil {
					return err
				}
			} else {
				sess, err = session.New(app.Config.Root, "", providerID, goal)
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "session: %s\n", sess.State.ID)
			}
			if wd, err := os.Getwd(); err == nil {
				if _, found, err := agentsdoc.Load(wd); err != nil {
					return err
				} else if found {
					fmt.Fprintf(os.Stderr, "loaded %s\n", agentsdoc.Path(wd))
				}
			}

			workDir, _ := os.Getwd()
			result, err := agent.Run(ctx, agent.RunRequest{
				Goal:       goal,
				ProviderID: providerID,
				SkillNames: skillNames,
				Session:    sess,
				Providers:  rt.Providers,
				Tools:      rt.Tools,
				Skills:        rt.Skills,
				SkillsCatalog: rt.SkillsCatalog,
				Config:     app.Config,
				RAG:        rt.RAG,
				Memory:     rt.Memory,
				MCP:              rt.MCP,
				DecisionProfiles: rt.DecisionProfiles,
				PackIndex:        rt.PackIndex,
				Hooks:            rt.Hooks,
				Questions:        rt.Questions,
				WorkDir:    workDir,
				EventHandler: func(ev events.Event) {
					if ev.Type != events.TypeThought {
						printEvent(os.Stderr, ev)
					}
				},
				StreamThoughtStart: func(n int) error {
					if n > 1 {
						fmt.Fprintln(os.Stderr)
					}
					fmt.Fprintf(os.Stderr, "taiji · thought %d\n", n)
					return nil
				},
				StreamThinkDelta: func(delta string) error {
					_, err := fmt.Fprint(os.Stderr, delta)
					return err
				},
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, result.Answer)
			return nil
		},
	}
	cmd.Flags().StringVar(&providerID, "provider", "", "provider id from config (default: default_provider)")
	cmd.Flags().StringArrayVar(&skillNames, "skill", nil, "skill name to load (repeatable)")
	cmd.Flags().StringVar(&sessionID, "session", "", "resume an existing session id")
	return cmd
}

func printEvent(w *os.File, ev events.Event) {
	switch ev.Type {
	case events.TypeStep:
		// UI chrome only; not sent to the model.
		return
	case events.TypeContinue:
		fmt.Fprintf(w, "[%s]\n", ev.Text)
	case events.TypeThought:
		fmt.Fprintf(w, "thinking (step %d): %s\n", ev.Step, ev.Text)
	case events.TypeMessage:
		if strings.TrimSpace(ev.Text) != "" {
			fmt.Fprintf(w, "assistant: %s\n", ev.Text)
		}
	case events.TypeToolCall:
		fmt.Fprintf(w, "tool %s(%s)\n", ev.ToolName, ev.Args)
	case events.TypeObservation:
		fmt.Fprintf(w, "observation %s: %s\n", ev.ToolName, ev.Text)
		if ev.Data != nil {
			if diffText, ok := ev.Data["diff"].(string); ok && strings.TrimSpace(diffText) != "" {
				fmt.Fprintf(w, "%s\n", diffText)
			}
		}
	case events.TypeMemoryRecalled, events.TypeMemoryStored, events.TypeMemoryRevised:
		if strings.TrimSpace(ev.Text) != "" {
			fmt.Fprintf(w, "%s: %s\n", ev.Type, ev.Text)
		}
	case events.TypeTurnMetrics:
		fmt.Fprintf(w, "metrics: %v\n", ev.Data)
	case events.TypeError:
		fmt.Fprintf(w, "error: %s\n", ev.Text)
	}
}
