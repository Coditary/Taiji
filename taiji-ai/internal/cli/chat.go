package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/coditary/taiji/internal/agent"
	"github.com/coditary/taiji/internal/agentsdoc"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/question"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/skills"
)

func newChatCmd(app *App) *cobra.Command {
	var providerID string
	var skillNames []string
	var sessionID string
	var sessionName string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Interactive agent chat (REPL)",
		Long: `Start an interactive chat session with the Taiji agent loop.

Type a goal or message and press Enter. Chained reasoning steps are printed to stderr as "taiji · thought N";
the final answer is printed to stdout under "taiji · answer". These labels are taiji UI chrome, not model output.

Slash commands:
  /exit, /quit     leave the chat
  /help            show commands
  /session         show the current session id
  /name [title]    show or set the session name
  /delete [id]     delete a session (current if omitted)
  /clear           start a new session
  /provider [id]   show or switch provider
  /tools           list available tools
  /skills          list invokable skills
  /compact         summarize older turns to save context`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !stdinIsTTY() {
				return fmt.Errorf("chat requires an interactive terminal (stdin must be a TTY)")
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return runChatREPL(ctx, app, chatOptions{
				providerID:   providerID,
				skillNames:   skillNames,
				sessionID:    sessionID,
				sessionName:  sessionName,
				runtimeOpts:  app.runtimeOptions(cmd),
			})
		},
	}
	cmd.Flags().StringVar(&providerID, "provider", "", "provider id from config (default: default_provider)")
	cmd.Flags().StringArrayVar(&skillNames, "skill", nil, "skill name to load (repeatable)")
	cmd.Flags().StringVar(&sessionID, "session", "", "resume an existing session id")
	cmd.Flags().StringVar(&sessionName, "name", "", "set a display name for this session")
	return cmd
}

type chatOptions struct {
	providerID  string
	skillNames  []string
	sessionID   string
	sessionName string
	runtimeOpts runtime.Options
}

type chatState struct {
	app        *App
	rt         *runtime.Runtime
	providerID string
	skillNames []string
	workDir    string
	session    *session.Session
}

func runChatREPL(parent context.Context, app *App, opts chatOptions) error {
	rt, err := runtime.New(parent, app.Config, opts.runtimeOpts)
	if err != nil {
		return err
	}
	defer rt.Close()
	rt.Questions.SetPrompter(question.CLIPrompter{In: os.Stdin, Out: os.Stderr})

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	state := &chatState{
		app:        app,
		rt:         rt,
		providerID: strings.TrimSpace(opts.providerID),
		skillNames: append([]string(nil), opts.skillNames...),
		workDir:    workDir,
	}

	resumed := false
	state.session, resumed, err = resolveChatSession(app, opts, workDir)
	if err != nil {
		return err
	}
	if state.providerID == "" && state.session != nil {
		state.providerID = state.session.State.Provider
	}

	if state.providerID == "" && state.session != nil {
		state.providerID = state.session.State.Provider
	}
	if name := strings.TrimSpace(opts.sessionName); name != "" {
		if err := state.session.SetName(name, true); err != nil {
			return err
		}
	}

	resolvedProvider, err := resolveChatProvider(state)
	if err != nil {
		return err
	}

	printChatBanner(os.Stderr, state.session, resolvedProvider, resumed)
	if _, found, err := agentsdoc.Load(workDir); err != nil {
		return err
	} else if found {
		fmt.Fprintf(os.Stderr, "loaded %s\n", agentsdoc.Path(workDir))
	}
	fmt.Fprintln(os.Stderr, "Type /help for commands. Ctrl+C cancels a run; Ctrl+D exits.")
	fmt.Fprintln(os.Stderr)

	reader := bufio.NewReader(os.Stdin)
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	var runCancel context.CancelFunc
	exitCh := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case <-exitCh:
				return
			case <-sigCh:
				if runCancel != nil {
					runCancel()
					fmt.Fprintln(os.Stderr, "\n(cancelled)")
					continue
				}
				fmt.Fprintln(os.Stderr, "\nbye")
				exitCh <- struct{}{}
				return
			}
		}
	}()

	for {
		select {
		case <-exitCh:
			return nil
		default:
		}

		fmt.Fprint(os.Stdout, ">>> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(os.Stderr, "bye")
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		handled, exit, err := state.handleSlashCommand(line)
		if err != nil {
			return err
		}
		if exit {
			fmt.Fprintln(os.Stderr, "bye")
			return nil
		}
		if handled {
			continue
		}

		runCtx, cancelRun := context.WithCancel(parent)
		runCancel = cancelRun
		streamed := false
		answerShown := false

		result, err := agent.Run(runCtx, agent.RunRequest{
			Goal:       line,
			ProviderID: state.providerID,
			SkillNames: state.skillNames,
			Session:    state.session,
			Providers:  state.rt.Providers,
			Tools:      state.rt.Tools,
			Skills:        state.rt.Skills,
			SkillsCatalog: state.rt.SkillsCatalog,
			Config:     state.app.Config,
			RAG:        state.rt.RAG,
			Memory:     state.rt.Memory,
			MCP:              state.rt.MCP,
			DecisionProfiles: state.rt.DecisionProfiles,
			PackIndex:        state.rt.PackIndex,
			Hooks:            state.rt.Hooks,
			Questions:        state.rt.Questions,
			WorkDir:    state.workDir,
			EventHandler: func(ev events.Event) {
				switch ev.Type {
				case events.TypeStep:
					answerShown = false
				case events.TypeThought:
					// streamed via StreamThinkDelta
				case events.TypeContinue:
					answerShown = false
				default:
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
			StreamDelta: func(delta string) error {
				if !answerShown {
					if !agent.IsVisibleAnswerDelta(delta) {
						return nil
					}
					fmt.Fprintln(os.Stderr)
					fmt.Fprintln(os.Stderr, "taiji · answer")
					answerShown = true
				}
				streamed = true
				_, werr := os.Stdout.WriteString(delta)
				return werr
			},
			StreamEnd: func() error {
				_, err := fmt.Fprintln(os.Stdout)
				return err
			},
		})
		runCancel = nil
		cancelRun()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
				fmt.Fprintln(os.Stderr, "(cancelled)")
				continue
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			fmt.Fprintln(os.Stderr)
			continue
		}
		if !answerShown && strings.TrimSpace(result.Answer) != "" {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "taiji · answer")
			fmt.Fprintln(os.Stdout, result.Answer)
		} else if strings.TrimSpace(result.Answer) != "" && !streamed {
			fmt.Fprintln(os.Stdout, result.Answer)
		}
		if err := maybeAutoNameSession(parent, state, line); err != nil {
			fmt.Fprintf(os.Stderr, "session naming: %v\n", err)
		}
		fmt.Fprintln(os.Stderr)
	}
}

func (s *chatState) handleSlashCommand(line string) (handled bool, exit bool, err error) {
	if !strings.HasPrefix(line, "/") {
		return false, false, nil
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return true, false, nil
	}

	if reply, ok, err := session.ParseCheckpointCommand(s.session, s.app.Config.Root, line); ok {
		if err != nil {
			return true, false, err
		}
		fmt.Fprintln(os.Stderr, reply)
		return true, false, nil
	}

	if strings.HasPrefix(strings.ToLower(line), "/council") {
		reply, ok, err := session.ParseCouncilCommand(s.session, s.app.Config.Root, line)
		if ok {
			if err != nil {
				return true, false, err
			}
			fmt.Fprintln(os.Stderr, reply)
			return true, false, nil
		}
	}

	switch strings.ToLower(fields[0]) {
	case "/exit", "/quit":
		return true, true, nil
	case "/help", "/?":
		printChatHelp(os.Stderr)
		return true, false, nil
	case "/session":
		fmt.Fprintf(os.Stderr, "session: %s\n", s.session.Label())
		st := s.session.State.Stats
		if st.Requests > 0 {
			fmt.Fprintf(os.Stderr, "  requests: %d  in: %d  out: %d  latency: %dms  cost: $%.4f\n",
				st.Requests, st.InputTokens, st.OutputTokens, st.TotalLatencyMs, st.EstimatedCostUSD)
		}
		return true, false, nil
	case "/name":
		if len(fields) == 1 {
			name := strings.TrimSpace(s.session.State.Name)
			if name == "" {
				fmt.Fprintln(os.Stderr, "session name: (unset)")
			} else {
				fmt.Fprintf(os.Stderr, "session name: %s\n", name)
			}
			return true, false, nil
		}
		name := strings.TrimSpace(strings.Join(fields[1:], " "))
		if name == "" {
			return true, false, fmt.Errorf("session name is required")
		}
		if err := s.session.SetName(name, true); err != nil {
			return true, false, err
		}
		fmt.Fprintf(os.Stderr, "session name set to %q\n", s.session.State.Name)
		return true, false, nil
	case "/delete":
		targetID := s.session.State.ID
		if len(fields) > 1 {
			targetID = strings.TrimSpace(fields[1])
		}
		if targetID == "" {
			return true, false, fmt.Errorf("session id is required")
		}
		deletingCurrent := targetID == s.session.State.ID
		if err := session.Delete(s.app.Config.Root, targetID); err != nil {
			return true, false, err
		}
		fmt.Fprintf(os.Stderr, "deleted session %q\n", targetID)
		if !deletingCurrent {
			return true, false, nil
		}
		s.session, err = session.New(s.app.Config.Root, "", s.providerID, "chat")
		if err != nil {
			return true, false, err
		}
		if err := session.SaveDirBinding(s.app.Config.Root, s.workDir, s.session.State.ID); err != nil {
			return true, false, err
		}
		fmt.Fprintf(os.Stderr, "new session: %s\n", s.session.Label())
		return true, false, nil
	case "/clear":
		s.session, err = session.New(s.app.Config.Root, "", s.providerID, "chat")
		if err != nil {
			return true, false, err
		}
		if err := session.SaveDirBinding(s.app.Config.Root, s.workDir, s.session.State.ID); err != nil {
			return true, false, err
		}
		fmt.Fprintf(os.Stderr, "new session: %s\n", s.session.Label())
		return true, false, nil
	case "/template", "/actor":
		if len(fields) < 2 {
			for _, t := range session.ListChatTemplates(s.app.Config.Root) {
				fmt.Fprintf(os.Stderr, "  %s — %s\n", t.ID, t.Name)
			}
			return true, false, nil
		}
		if err := session.ApplyChatTemplate(s.session, s.app.Config.Root, fields[1]); err != nil {
			return true, false, err
		}
		fmt.Fprintf(os.Stderr, "template active: %s\n", fields[1])
		return true, false, nil
	case "/provider":
		if len(fields) == 1 {
			name, err := resolveChatProvider(s)
			if err != nil {
				return true, false, err
			}
			fmt.Fprintf(os.Stderr, "provider: %s\n", name)
			return true, false, nil
		}
		id := strings.TrimSpace(fields[1])
		if id == "" {
			return true, false, fmt.Errorf("provider id is required")
		}
		if _, err := s.rt.Providers.Get(id); err != nil {
			return true, false, err
		}
		s.providerID = id
		fmt.Fprintf(os.Stderr, "provider set to %q\n", id)
		return true, false, nil
	case "/tools":
		defs := s.rt.Tools.Definitions()
		sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
		for _, def := range defs {
			desc := strings.TrimSpace(def.Description)
			if desc == "" {
				fmt.Fprintf(os.Stderr, "- %s\n", def.Name)
				continue
			}
			fmt.Fprintf(os.Stderr, "- %s: %s\n", def.Name, desc)
		}
		if len(defs) == 0 {
			fmt.Fprintln(os.Stderr, "(no tools registered)")
		}
		return true, false, nil
	case "/compact":
		summary, err := agent.CompactSessionHistory(context.Background(), s.session, s.rt.Providers, s.providerID, s.rt.Config)
		if err != nil {
			return true, false, err
		}
		preview := strings.TrimSpace(summary)
		if len(preview) > 200 {
			preview = preview[:200] + "…"
		}
		fmt.Fprintf(os.Stderr, "compacted session %q\n%s\n", s.session.Label(), preview)
		return true, false, nil
	case "/skills":
		if s.rt.Skills == nil {
			fmt.Fprintln(os.Stderr, "(skills not configured)")
			return true, false, nil
		}
		all, err := s.rt.Skills.LoadAll()
		if err != nil {
			return true, false, err
		}
		invokable := skills.Invokable(all)
		sort.Slice(invokable, func(i, j int) bool { return invokable[i].Name < invokable[j].Name })
		for _, skill := range invokable {
			desc := strings.TrimSpace(skill.Description)
			if desc == "" {
				fmt.Fprintf(os.Stderr, "- %s\n", skill.Name)
				continue
			}
			fmt.Fprintf(os.Stderr, "- %s: %s\n", skill.Name, desc)
		}
		if len(invokable) == 0 {
			fmt.Fprintln(os.Stderr, "(no invokable skills)")
		}
		return true, false, nil
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (try /help)\n", fields[0])
		return true, false, nil
	}
}

func printChatBanner(w io.Writer, sess *session.Session, providerName string, resumed bool) {
	if resumed {
		fmt.Fprintf(w, "taiji chat (resumed session: %s, provider: %s)\n", sess.Label(), providerName)
		return
	}
	fmt.Fprintf(w, "taiji chat (session: %s, provider: %s)\n", sess.Label(), providerName)
}

func maybeAutoNameSession(ctx context.Context, state *chatState, userMessage string) error {
	if state.app.Config == nil || !state.app.Config.Agent.AutoNameSessionsEnabled() {
		return nil
	}
	if !state.session.NeedsAutoName() {
		return nil
	}
	name, err := session.AutoName(ctx, state.session, state.rt.Providers, state.providerID, userMessage)
	if err != nil {
		return err
	}
	if name != "" {
		fmt.Fprintf(os.Stderr, "(session named: %q)\n", name)
	}
	return nil
}

func resolveChatSession(app *App, opts chatOptions, workDir string) (*session.Session, bool, error) {
	root := app.Config.Root
	if opts.sessionID != "" {
		sess, err := session.Load(root, opts.sessionID)
		if err != nil {
			return nil, false, err
		}
		if err := session.SaveDirBinding(root, workDir, sess.State.ID); err != nil {
			return nil, false, err
		}
		return sess, true, nil
	}

	if binding, ok, err := session.LoadDirBinding(root, workDir); err != nil {
		return nil, false, err
	} else if ok {
		sess, err := session.Load(root, binding.SessionID)
		if err == nil {
			return sess, true, nil
		}
	}

	sess, err := session.New(root, "", strings.TrimSpace(opts.providerID), "chat")
	if err != nil {
		return nil, false, err
	}
	if err := session.SaveDirBinding(root, workDir, sess.State.ID); err != nil {
		return nil, false, err
	}
	return sess, false, nil
}

func resolveChatProvider(s *chatState) (string, error) {
	if strings.TrimSpace(s.providerID) != "" {
		p, err := s.rt.Providers.Get(s.providerID)
		if err != nil {
			return "", err
		}
		return p.ID(), nil
	}
	p, err := s.rt.Providers.Default()
	if err != nil {
		return "", err
	}
	s.providerID = p.ID()
	return p.ID(), nil
}

func printChatHelp(w io.Writer) {
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  /exit, /quit     leave the chat")
	fmt.Fprintln(w, "  /help            show this help")
	fmt.Fprintln(w, "  /session         show session id and usage stats")
	fmt.Fprintln(w, "  /name [title]    show or set the session name")
	fmt.Fprintln(w, "  /delete [id]     delete a session (current if omitted)")
	fmt.Fprintln(w, "  /clear           start a new session")
	fmt.Fprintln(w, "  /provider [id]   show or switch provider")
	fmt.Fprintln(w, "  /tools           list available tools")
	fmt.Fprintln(w, "  /skills          list invokable skills")
	fmt.Fprintln(w, "  /compact         summarize older turns to save context")
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
