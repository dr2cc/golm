package lmcz

import (
	"context"
	"fmt"

	"github.com/dr2cc/golm/internal/report"
)

// [GUI-статусы] Шаги проверки ЛМ ЧЗ для отображения в GUI.
// Здесь только обертки: они вызывают прежние resolveHostPort / statusAt / sendToken
// и сообщают Tracker'у, чем закончился каждый шаг. Сама логика запросов не менялась.
// В CLI onStep == nil, Tracker ничего не делает, вывод в консоль прежний.

const (
	StepFind   = "find"
	StepStatus = "status"
	StepToken  = "token"
)

var (
	stepFind   = report.Step{ID: StepFind, Title: "Поиск сервера ЛМ ЧЗ"}
	stepStatus = report.Step{ID: StepStatus, Title: "Статус сервера"}
	stepToken  = report.Step{ID: StepToken, Title: "Отправка токена"}
)

// findServer — шаг «Поиск сервера»: адрес из настроек или найденный сканированием подсети.
func (c *CzClient) findServer(steps *report.Tracker) (string, error) {
	steps.Start(StepFind)
	hostPort, err := c.resolveHostPort()
	if err != nil {
		steps.Done(StepFind, report.Fail, err.Error())
		steps.SkipRest("не выполнялся: сервер не найден")
		return "", err
	}

	detail := hostPort + " (из настроек)"
	if c.cfg.Host == "" {
		detail = hostPort + " (найден сканированием подсети)"
	}
	steps.Done(StepFind, report.OK, detail)
	return hostPort, nil
}

// queryStatus — шаг «Статус сервера»: GET /api/v2/status по найденному адресу.
func (c *CzClient) queryStatus(ctx context.Context, steps *report.Tracker, hostPort string) (CheckResult, error) {
	steps.Start(StepStatus)
	res, err := c.statusAt(ctx, hostPort)
	if err != nil {
		steps.Done(StepStatus, report.Fail, err.Error())
		steps.SkipRest("не выполнялся: статус не получен")
		return res, err
	}
	steps.Done(StepStatus, report.OK, fmt.Sprintf("(%d) %s", res.HttpCode, res.Status))
	return res, nil
}
