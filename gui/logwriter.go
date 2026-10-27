package main

import (
	"bytes"
	"sync"
)

// lineWriter — io.Writer, который передается клиентам golm через SetOutput.
// Клиенты пишут в него ровно те же строки, что CLI печатает в консоль,
// а lineWriter режет поток на строки и отдает каждую в emit (событие "log" для фронтенда).
//
// Запись может идти из нескольких горутин одновременно (сканер подсети),
// поэтому все операции защищены мьютексом.
type lineWriter struct {
	mu   sync.Mutex
	buf  []byte
	emit func(line string)
}

func newLineWriter(emit func(string)) *lineWriter {
	return &lineWriter{emit: emit}
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := string(bytes.TrimRight(w.buf[:i], "\r"))
		w.buf = w.buf[i+1:]
		w.emit(line)
	}
	return len(p), nil
}

// Flush отдает остаток без завершающего перевода строки
// (например, "Успех (200)!" печатается без \n).
func (w *lineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buf) > 0 {
		w.emit(string(w.buf))
		w.buf = nil
	}
}
