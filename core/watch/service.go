package watch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/kimi-sdk-agent-indexer/core/buffer"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/prompts"
	"github.com/user/kimi-sdk-agent-indexer/core/logging"
	"github.com/user/kimi-sdk-agent-indexer/core/validator"
	"github.com/user/kimi-sdk-agent-indexer/core/watcher"
)

// Session abstracts the LLM conversation.
type Session interface {
	Send(ctx context.Context, prompt string) (string, error)
}

// SessionFactory creates sessions for each batch.
type SessionFactory interface {
	NewSession(ctx context.Context) (Session, error)
}

// Config controls the watch service behaviour.
type Config struct {
	WorkspaceRoot   string
	IndexPath       string
	SchemaDir       string
	StorageSpecPath string
	MaxIterations   int
}

// Service wires watcher, buffer, validator, and LLM session into the watch loop.
type Service struct {
	cfg      Config
	watcher  *watcher.Watcher
	buffer   *buffer.Buffer
	sessions SessionFactory
	log      logging.Printer
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewService constructs the watch service.
func NewService(cfg Config, w *watcher.Watcher, buf *buffer.Buffer, sessions SessionFactory, log logging.Printer) (*Service, error) {
	if cfg.WorkspaceRoot == "" {
		return nil, errors.New("watch: workspace root required")
	}
	if cfg.IndexPath == "" {
		return nil, errors.New("watch: index path required")
	}
	if cfg.SchemaDir == "" {
		return nil, errors.New("watch: schema dir required")
	}
	if cfg.StorageSpecPath == "" {
		cfg.StorageSpecPath = "docs/design/storage-design.md"
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 100
	}

	if w == nil {
		return nil, errors.New("watch: watcher is nil")
	}
	if buf == nil {
		buf = buffer.New()
	}
	if sessions == nil {
		return nil, errors.New("watch: session factory required")
	}
	if log == nil {
		log = logging.New().WithComponent("watch")
	} else {
		log = log.WithComponent("watch")
	}

	return &Service{
		cfg:      cfg,
		watcher:  w,
		buffer:   buf,
		sessions: sessions,
		log:      log,
	}, nil
}

// Run starts the watcher loop and blocks until ctx is done.
func (s *Service) Run(ctx context.Context) error {
	if err := s.watcher.Start(); err != nil {
		return err
	}
	defer s.watcher.Stop()

	// Relay watcher events into the buffer.
	go func() {
		for ev := range s.watcher.Events() {
			s.buffer.Ingest(buffer.SourceEvent{
				Path: ev.Path,
				Op:   string(ev.Op),
			})
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.buffer.NotifyChan():
			s.handleBatch(ctx)
		}
	}
}

func (s *Service) handleBatch(ctx context.Context) {
	changes := s.buffer.Flush()
	if len(changes) == 0 {
		return
	}

	session, err := s.sessions.NewSession(ctx)
	if err != nil {
		s.log.Errorf("watch: new session: %v", err)
		return
	}

	batchID := newBatchID()

	data := prompts.WatchTemplateData{
		WorkspaceRoot:     s.cfg.WorkspaceRoot,
		ChangeBatchID:     batchID,
		ChangedFiles:      renderChangedFiles(s.cfg.WorkspaceRoot, changes),
		ChangedFileBlobs:  renderChangedBlobs(s.cfg.WorkspaceRoot, changes),
		RelatedIndexFiles: renderIndexFiles(s.cfg.IndexPath),
		StorageSpecPath:   s.cfg.StorageSpecPath,
	}

	prompt, err := prompts.RenderWatch(data)
	if err != nil {
		s.log.Errorf("watch: render prompt: %v", err)
		return
	}

	if _, err := session.Send(ctx, prompt); err != nil {
		s.log.Errorf("watch: session send: %v", err)
		return
	}

	val, err := validator.New(validator.Config{
		IndexPath: s.cfg.IndexPath,
		SchemaDir: s.cfg.SchemaDir,
	})
	if err != nil {
		s.log.Errorf("watch: validator init: %v", err)
		return
	}

	for attempt := 1; attempt <= s.cfg.MaxIterations; attempt++ {
		if err := val.Validate(ctx); err != nil {
			if attempt == s.cfg.MaxIterations {
				s.log.Errorf("watch: validation failed after %d attempts: %v", attempt, err)
				return
			}
			feedback, ferr := prompts.AppendValidatorFeedback(prompts.ValidatorFeedbackData{
				Attempt: attempt + 1,
				Error:   err.Error(),
			})
			if ferr != nil {
				s.log.Errorf("watch: render feedback: %v", ferr)
				return
			}
			if _, err = session.Send(ctx, feedback); err != nil {
				s.log.Errorf("watch: session retry: %v", err)
				return
			}
			continue
		}

		s.log.Infof("watch: batch %s applied (%d changes)", batchID, len(changes))
		return
	}
}

func renderChangedFiles(root string, changes []buffer.Change) string {
	if len(changes) == 0 {
		return "No changes."
	}
	var b strings.Builder
	for _, change := range changes {
		rel := makeRelative(root, change.Path)
		fmt.Fprintf(&b, "- %s (%s)\n", rel, change.Kind)
	}
	return b.String()
}

func renderChangedBlobs(root string, changes []buffer.Change) string {
	if len(changes) == 0 {
		return "[]"
	}
	var b strings.Builder
	for _, change := range changes {
		rel := makeRelative(root, change.Path)
		b.WriteString("```")
		b.WriteString("\n")
		b.WriteString("# ")
		b.WriteString(rel)
		b.WriteString(" (")
		b.WriteString(change.Kind.String())
		b.WriteString(")\n")
		content, err := os.ReadFile(change.Path)
		if err != nil {
			fmt.Fprintf(&b, "[unavailable: %v]\n", err)
		} else {
			b.Write(content)
		}
		if len(content) == 0 || content[len(content)-1] != '\n' {
			b.WriteByte('\n')
		}
		b.WriteString("```\n")
	}
	return b.String()
}

func renderIndexFiles(indexRoot string) string {
	var entries []string
	filepath.WalkDir(indexRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(indexRoot, path)
		bytes, readErr := os.ReadFile(path)
		if readErr != nil {
			entries = append(entries, fmt.Sprintf("- %s (error: %v)", filepath.ToSlash(rel), readErr))
			return nil
		}
		entries = append(entries, fmt.Sprintf("## %s\n```\n%s\n```", filepath.ToSlash(rel), string(bytes)))
		return nil
	})
	return strings.Join(entries, "\n")
}

func makeRelative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func newBatchID() string {
	return fmt.Sprintf("%d-%06d", time.Now().UnixNano(), rand.Intn(1_000_000))
}
