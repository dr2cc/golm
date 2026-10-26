// Package report описывает шаги проверки и их состояния.
//
// [GUI-статусы] Пакет добавлен для GUI: окну нужно знать не только общий итог
// проверки, но и что произошло на каждом шаге (Apache, модуль 1С, сервис DataMobile...).
// Клиенты сообщают о шагах через Func. В CLI Func не задается (nil),
// поэтому все вызовы Tracker там ничего не делают и консольный вывод не меняется.
package report

// State — состояние шага.
type State string

const (
	Pending State = "pending" // шаг запланирован, еще не начат
	Running State = "running" // выполняется
	OK      State = "ok"      // успешно
	Warn    State = "warn"    // выполнен, но есть замечание (например, медленный ответ)
	Fail    State = "fail"    // ошибка
	Skipped State = "skipped" // не выполнялся (не нужен или прерван ошибкой на предыдущем шаге)
)

// Step — снимок состояния одного шага. Отправляется получателю при каждом изменении.
type Step struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	State  State  `json:"state"`
	Detail string `json:"detail"`
}

// Func — получатель обновлений шагов (в GUI — отправка события во фронтенд).
type Func func(Step)

// Tracker хранит список шагов одной проверки и рассылает их изменения.
// Все методы безопасно вызывать при emit == nil — тогда они ничего не делают.
type Tracker struct {
	emit  Func
	steps []Step
}

// NewTracker сразу сообщает весь план проверки (все шаги в состоянии Pending),
// чтобы окно показало список шагов еще до начала работы.
func NewTracker(emit Func, steps ...Step) *Tracker {
	t := &Tracker{emit: emit, steps: steps}
	for i := range t.steps {
		t.steps[i].State = Pending
		t.send(i)
	}
	return t
}

// Start отмечает шаг как выполняющийся.
func (t *Tracker) Start(id string) {
	t.set(id, Running, "")
}

// Done завершает шаг с указанным состоянием и пояснением.
func (t *Tracker) Done(id string, state State, detail string) {
	t.set(id, state, detail)
}

// SkipRest отмечает все еще не начатые шаги как пропущенные.
// Вызывается, когда проверка прервалась на одном из шагов.
func (t *Tracker) SkipRest(detail string) {
	for i := range t.steps {
		if t.steps[i].State == Pending {
			t.steps[i].State = Skipped
			t.steps[i].Detail = detail
			t.send(i)
		}
	}
}

func (t *Tracker) set(id string, state State, detail string) {
	for i := range t.steps {
		if t.steps[i].ID == id {
			t.steps[i].State = state
			t.steps[i].Detail = detail
			t.send(i)
			return
		}
	}
}

func (t *Tracker) send(i int) {
	if t.emit != nil {
		t.emit(t.steps[i])
	}
}
