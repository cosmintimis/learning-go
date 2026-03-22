package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// MsgLogger writes received-message logs and error logs to separate files.
type MsgLogger struct {
	msgFile *os.File
	errFile *os.File
	msgLog  *log.Logger
	errLog  *log.Logger
}

const logsDir = "logs"

// NewMsgLogger creates a "logs" directory (if it doesn't exist) and opens
// logs/node_<index>_messages.log and logs/node_<index>_errors.log inside it.
// Caller must call Close() when done.
func NewMsgLogger(nodeIndex int) (*MsgLogger, error) {
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, fmt.Errorf("NewMsgLogger: create logs dir: %w", err)
	}

	msgPath := filepath.Join(logsDir, fmt.Sprintf("node_%d_messages.log", nodeIndex))
	errPath := filepath.Join(logsDir, fmt.Sprintf("node_%d_errors.log", nodeIndex))

	msgFile, err := os.Create(msgPath)
	if err != nil {
		return nil, fmt.Errorf("NewMsgLogger: create %s: %w", msgPath, err)
	}
	errFile, err := os.Create(errPath)
	if err != nil {
		msgFile.Close()
		return nil, fmt.Errorf("NewMsgLogger: create %s: %w", errPath, err)
	}

	return &MsgLogger{
		msgFile: msgFile,
		errFile: errFile,
		msgLog:  log.New(msgFile, "", 0),
		errLog:  log.New(errFile, "", log.LstdFlags),
	}, nil
}

// LogMessage writes one line: "OK/FAIL <source_index> <sent_sha1_hex> <calc_sha1_hex>"
func (l *MsgLogger) LogMessage(ok bool, sourceIndex uint8, sentHex, calcHex string) {
	status := "OK"
	if !ok {
		status = "FAIL"
	}
	l.msgLog.Printf("%s %d %s %s", status, sourceIndex, sentHex, calcHex)
}

// LogError writes a formatted error line to the error log file.
func (l *MsgLogger) LogError(format string, args ...any) {
	l.errLog.Printf(format, args...)
}

// Close flushes and closes both log files.
func (l *MsgLogger) Close() {
	l.msgFile.Close()
	l.errFile.Close()
}
