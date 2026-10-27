package lmcz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/dr2cc/golm/internal/report"
)

// Init — публичный метод. Он координирует процесс: проверяет статус
// и при необходимости выполняет инициализацию, переиспользуя логику Check.
func (c *CzClient) Init(ctx context.Context) error {
	// 1. Делаем предварительную проверку статуса.
	// Метод Check сам разберется: сканировать сеть или брать жесткий Host,
	// очистит префиксы, сделает GET-запрос и вернет строковый статус.
	//
	// [GUI-рефакторинг] Вместо c.Status(ctx) вызываем его две части по отдельности:
	// адрес находим один раз (resolveHostPort) и запоминаем в hostPort,
	// а затем используем его и для проверки статуса, и для отправки токена (шаг 3).
	// Раньше на шаге 3 вызывался getTargetHostPort(), который при пустом Host
	// сканировал подсеть второй раз. Текст ошибок для пользователя остался прежним.
	//
	// [GUI-статусы] Части вызываются через обертки findServer / queryStatus (steps.go),
	// чтобы GUI видел итог каждого шага. В CLI шаги никуда не отправляются.
	steps := report.NewTracker(c.onStep, stepFind, stepStatus, stepToken)

	hostPort, err := c.findServer(steps)
	if err != nil {
		return fmt.Errorf("предварительная проверка статуса перед инициализацией неудачна: %w", err)
	}

	checkResult, err := c.queryStatus(ctx, steps, hostPort)
	if err != nil {
		return fmt.Errorf("предварительная проверка статуса перед инициализацией неудачна: %w", err)
	}

	// [GUI-рефакторинг] Здесь и ниже fmt.Printf/Println заменены на fmt.Fprintf/Fprintln(c.out, ...),
	// чтобы вывод можно было перенаправить в GUI. В CLI c.out == os.Stdout, текст тот же.
	fmt.Fprintf(c.out, "LMCZ server response: (%d) status: %s\n", checkResult.HttpCode, checkResult.Status)

	// 2. Бизнес-логика: отправляем токен только если сервер не настроен
	if checkResult.Status != "not_configured" {
		fmt.Fprintln(c.out, "Сервер LMCZ уже настроен. Отправка токена инициализации не требуется.")
		steps.Done(StepToken, report.Skipped, "не требуется: сервер уже настроен")
		return nil
	}

	fmt.Fprintln(c.out, "Server not configured. Sending token....")

	// 3. Вызываем приватный метод для отправки токена.
	// [GUI-рефакторинг] hostPort уже найден на шаге 1 — повторного сканирования нет.
	steps.Start(StepToken)
	if err := c.sendToken(ctx, hostPort); err != nil {
		steps.Done(StepToken, report.Fail, err.Error())
		return err
	}
	steps.Done(StepToken, report.OK, "токен принят сервером")

	return nil
}

// sendToken — приватный метод. Отвечает строго за техническую
// отправку POST-запроса на конкретный хост.
func (c *CzClient) sendToken(ctx context.Context, hostPort string) error {
	payload := InitRequestPayload{Token: c.cfg.TokenXAPIKEY}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	targetURL := fmt.Sprintf("http://%s/api/v2/init", hostPort)

	// Формируем запрос:
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.cfg.User, c.cfg.Pass)

	// Обрабатывем ответ:
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Распаковываем ошибку http-клиента, теперь это работает без ошибок компилятора
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return fmt.Errorf("сервер недоступен по адресу %s: %w", hostPort, urlErr.Err)
		}
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем HTTP статус-код (все что вне диапазона 2xx — ошибка)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr InitErrorPayload

		// Пытаемся прочитать JSON-ответ от сервера, чтобы узнать точную причину ошибки
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Reason != "" {
			// Оборачиваем apiErr. Названия полей (errorCode, reason)
			// подставятся автоматически благодаря методу Error() у InitErrorPayload
			return fmt.Errorf("(%d) %w", resp.StatusCode, apiErr)
		}

		return fmt.Errorf("unexpected status code: (%d) %w", resp.StatusCode, apiErr)
		// // Если сервер прислал не JSON (например, ошибку nginx), отдаем стандартный статус
		// return fmt.Errorf("unexpected status codee: %d (%s)", resp.StatusCode, resp.Status)
	}

	// Если статус 2xx — успешно декодируем финальный ответ
	var result InitResponsePayload
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// При успешной инициализации возвращается пустое тело и StatusOK,
		if errors.Is(err, io.EOF) {
			fmt.Fprintf(c.out, "Успех (%d)!", resp.StatusCode)
			return nil
		}
		// Вдруг что-то еще..
		return fmt.Errorf("failed to decode JSON response (HTTP %d): %w", resp.StatusCode, err)
	}

	fmt.Fprintf(c.out, "Ответ сервера на инициализацию: %s (Сообщение: %s)\n", result.Status, result.Message)
	return nil
}

// [GUI-рефакторинг] Здесь был метод getTargetHostPort(). Он удален:
// - при пустом Host он запускал ScanSubnet второй раз (сначала в Status, потом здесь);
// - при заданном Host дублировал urlutil.JoinHostPort, но без порта по умолчанию.
// Теперь Init получает адрес один раз через resolveHostPort() (см. status.go).
