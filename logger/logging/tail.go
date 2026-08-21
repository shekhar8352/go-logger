package logging

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
	"time"
)

// TailOptions configures TailLogs (follow new lines, similar to tail -f).
type TailOptions struct {
	Config    LoggerConfig
	Date      string
	PollEvery time.Duration
	// FromStart, when true, replays existing lines before following new ones.
	FromStart bool
}

// TailLogs follows the dated log file and sends newly parsed entries on the
// returned channel. The channel is closed when ctx is cancelled. Existing
// Search/Read APIs are unchanged; this only observes new lines by default.
func TailLogs(ctx context.Context, opts TailOptions) (<-chan LogEntry, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	dir, prefix := readerDirAndPrefix(opts.Config)
	date := opts.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	poll := opts.PollEvery
	if poll <= 0 {
		poll = 50 * time.Millisecond
	}
	path := datedLogFile(dir, prefix, date)
	ch := make(chan LogEntry, 32)
	go func() {
		defer close(ch)
		followLogFile(ctx, path, poll, opts.FromStart, ch)
	}()
	return ch, nil
}

func followLogFile(ctx context.Context, path string, poll time.Duration, fromStart bool, ch chan<- LogEntry) {
	var file *os.File
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	open := func() error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if file != nil {
			_ = file.Close()
		}
		file = f
		if !fromStart {
			if info, statErr := f.Stat(); statErr == nil {
				_, _ = f.Seek(info.Size(), io.SeekStart)
			}
			fromStart = true
		}
		return nil
	}

	for file == nil {
		if err := open(); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(poll):
		}
	}

	reader := bufio.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(poll):
			}
			info, statErr := file.Stat()
			if statErr != nil {
				if openErr := open(); openErr == nil {
					reader.Reset(file)
				}
				continue
			}
			pos, _ := file.Seek(0, io.SeekCurrent)
			if info.Size() < pos {
				_, _ = file.Seek(0, io.SeekStart)
				reader.Reset(file)
			}
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry, parseErr := ParseLogLine(line)
		if parseErr != nil {
			continue
		}
		select {
		case ch <- entry:
		case <-ctx.Done():
			return
		}
	}
}
