package logstream

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const consoleReadBufferBytes = 32 << 10

type consoleCapture struct {
	reader   *os.File
	original *os.File
	source   string
}

type consoleLineFramer struct {
	hub       *Hub
	source    string
	line      []byte
	truncated bool
}

var (
	consoleCaptureMu      sync.Mutex
	consoleCaptureStarted bool
)

// StartConsoleCapture 接管当前进程标准输出和标准错误，并在原样回写控制台的同时发布实时日志。
func StartConsoleCapture(hub *Hub) error {
	if hub == nil {
		return fmt.Errorf("运行日志中心不能为空")
	}
	consoleCaptureMu.Lock()
	defer consoleCaptureMu.Unlock()
	if consoleCaptureStarted {
		return nil
	}
	stdoutCapture, err := redirectConsoleOutput(os.Stdout, "stdout")
	if err != nil {
		return fmt.Errorf("接管标准输出失败: %w", err)
	}
	var stderrCapture *consoleCapture
	stderrCapture, err = redirectConsoleOutput(os.Stderr, "stderr")
	if err != nil {
		restoreConsoleOutput(os.Stdout, stdoutCapture)
		return fmt.Errorf("接管标准错误失败: %w", err)
	}
	consoleCaptureStarted = true
	go relayConsoleOutput(stdoutCapture, hub)
	go relayConsoleOutput(stderrCapture, hub)
	return nil
}

// redirectConsoleOutput 将目标文件描述符重定向到管道，并保留原输出用于回写。
func redirectConsoleOutput(target *os.File, source string) (*consoleCapture, error) {
	originalFD, err := unix.Dup(int(target.Fd()))
	if err != nil {
		return nil, err
	}
	original := os.NewFile(uintptr(originalFD), source+"-original")
	var reader *os.File
	var writer *os.File
	reader, writer, err = os.Pipe()
	if err != nil {
		_ = original.Close()
		return nil, err
	}
	if err = unix.Dup2(int(writer.Fd()), int(target.Fd())); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		_ = original.Close()
		return nil, err
	}
	if err = writer.Close(); err != nil {
		_ = unix.Dup2(int(original.Fd()), int(target.Fd()))
		_ = reader.Close()
		_ = original.Close()
		return nil, err
	}
	return &consoleCapture{reader: reader, original: original, source: source}, nil
}

// restoreConsoleOutput 恢复接管失败前的控制台输出。
func restoreConsoleOutput(target *os.File, capture *consoleCapture) {
	if capture == nil {
		return
	}
	_ = unix.Dup2(int(capture.original.Fd()), int(target.Fd()))
	_ = capture.reader.Close()
	_ = capture.original.Close()
}

// relayConsoleOutput 持续转发控制台字节，并将完整行写入运行日志中心。
func relayConsoleOutput(capture *consoleCapture, hub *Hub) {
	framer := &consoleLineFramer{
		hub:    hub,
		source: capture.source,
		line:   make([]byte, 0, consoleReadBufferBytes),
	}
	buffer := make([]byte, consoleReadBufferBytes)
	for {
		n, err := capture.reader.Read(buffer)
		if n > 0 {
			writeConsoleOriginal(capture.original, buffer[:n])
			framer.consume(buffer[:n])
		}
		if err != nil {
			framer.flush()
			return
		}
	}
}

// writeConsoleOriginal 尽力将接管到的字节完整写回原控制台。
func writeConsoleOriginal(original *os.File, data []byte) {
	for len(data) > 0 {
		n, err := original.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil || n == 0 {
			return
		}
	}
}

// consume 按换行符切分任意格式的控制台内容。
func (f *consoleLineFramer) consume(data []byte) {
	for len(data) > 0 {
		index := bytes.IndexByte(data, '\n')
		if index < 0 {
			f.appendFragment(data)
			return
		}
		f.appendFragment(data[:index])
		f.emit()
		data = data[index+1:]
	}
}

// appendFragment 在固定内存上限内保存当前日志行。
func (f *consoleLineFramer) appendFragment(fragment []byte) {
	remaining := maxEntryLineBytes - len(f.line)
	if remaining > 0 {
		f.line = append(f.line, fragment[:min(len(fragment), remaining)]...)
	}
	if len(fragment) > remaining {
		f.truncated = true
	}
}

// emit 发布当前完整日志行并重置分帧状态。
func (f *consoleLineFramer) emit() {
	line := strings.TrimSuffix(strings.ToValidUTF8(string(f.line), "\uFFFD"), "\r")
	entry := ParseLine(line, f.truncated)
	if entry.GetSource() == "" {
		entry.Source = f.source
	}
	f.hub.Append(entry)
	f.line = f.line[:0]
	f.truncated = false
}

// flush 在控制台流关闭时发布尚未换行的剩余内容。
func (f *consoleLineFramer) flush() {
	if len(f.line) > 0 || f.truncated {
		f.emit()
	}
}
