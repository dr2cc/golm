package lmcz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/scanner"
)

// Client инкапсулирует настройки для работы с удаленным сервером.
// Хранение http.Client внутри структуры — это Go-идиома, позволяющая переиспользовать соединения.
type Client struct {
	cfg        config.LmczConfig
	httpClient *http.Client
}

// New создает и возвращает настроенный экземпляр Client.
// Мы явно передаем таймаут, избегая глобальных дефолтов.
func NewClient(lmcz config.LmczConfig, httpClient *http.Client) *Client {
	return &Client{
		cfg:        lmcz,
		httpClient: httpClient,
	}
}

func (c *Client) Check(ctx context.Context) (string, error) {
	var hostPort string

	if c.cfg.Host == "" {
		// Находим компьютеры с портом сервиса ЛМ ЧЗ и получаем первый результат
		hostPort = scanner.ScanSubnet(c.cfg.Subnet, c.cfg.Port, c.cfg.ScannerTimeout)
	} else {
		// Очищаем хост от схемы на случай, если в конфиге указали "http://..."
		cleanHost := strings.TrimPrefix(c.cfg.Host, "http://")
		cleanHost = strings.TrimPrefix(cleanHost, "https://")
		hostPort = net.JoinHostPort(cleanHost, c.cfg.Port)
	}

	if hostPort == "" {
		return "", fmt.Errorf("хостов с открытым портом %s не найдено", c.cfg.Port)
	}

	// Добавляем схему
	finalURL := fmt.Sprintf("http://%s/api/v2/status", hostPort)

	fmt.Printf("Проверяем статус ЛМ ЧЗ по адресу: %s\n", finalURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, finalURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	// Формат ответа в рамках HTTP это *Response
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Распаковываем ошибку http-клиента, чтобы убрать дублирование URL
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return "", fmt.Errorf("сервер LMCZ недоступен по адресу %s: %w", hostPort, urlErr.Err)
		}
		return "", fmt.Errorf("запрос к LMCZ завершился ошибкой: %w", err)
	}
	// ЗОЛОТОЕ ПРАВИЛО Go для HTTP-запросов:
	// МОЖНО обращаться к resp.Body (ниже) ТОЛЬКО в том случае, если err == nil.

	// КАЖДЫЙ РАЗ, когда успешно получен http.Response (err == nil и resp != nil),
	// нужно вызвать resp.Body.Close() (регистрируем закрытие тела)
	defer resp.Body.Close()

	// Проверяем статус-код ответа (Чистый код: сервер ответил, но статус плохой)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("сервер LMCZ вернул статус: %s", resp.Status)
	}

	// 📌 Считываем содержимое тела. Три популярных метода:

	// 1. Получаем всё как строку или байты (маленькие ответы, JSON, HTML)
	b, err := io.ReadAll(resp.Body)
	// Так как "свиток" (scroll) resp.Body это однонаправленный поток (stream), не возможно «перемотать» его назад.
	// Как только данные будут прочитаны (например, с помощью io.ReadAll(resp.Body)), повторное чтение вернет io.EOF (конец файла).
	if err != nil {
		return "", fmt.Errorf("failed to read Response.Body: %w", err)
	}
	// // 2. Парсим JSON (самый эффективный способ)
	// var user UserStruct
	// err := json.NewDecoder(resp.Body).Decode(&user)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// // 3. Сохранение большого файла на диск
	// out, err := os.Create("downloaded_file.zip")
	// if err != nil {
	//     log.Fatal(err)
	// }
	// defer out.Close()
	// // Данные перетекают из сети в файл буферизованно
	// _, err = io.Copy(out, resp.Body)
	//

	// Отправляем содержимое тела ответа в стандартный поток вывода
	// Коротко про "стандартный поток вывода" (stdout).
	// В операционных системах (Linux, Windows, macOS)
	// у каждой программы при старте есть три стандартных потока данных:
	// - Стандартный ввод (stdin) — то, что пользователь вводит с клавиатуры в консоль.
	// - Стандартный вывод (stdout) — то, куда программа выводит свой обычный текст.
	// fmt.Printf и fmt.Println по умолчанию пишут именно сюда. Вы видите этот текст прямо в терминале.
	// - Стандартный вывод ошибок (stderr) — специальный отдельный поток для ошибок. Туда пишет, например, log.Println.

	// Парсим статус из ответа
	var statusResp StatusResponse
	if err := json.Unmarshal(b, &statusResp); err != nil {
		// Если сервер вернул не JSON, но ответил 200 OK,
		// возвращаем сырой текст как статус (на всякий случай)
		return string(b), nil
	}

	return statusResp.Status, nil
}

// Init — публичный метод. Он оркеструет процесс: проверяет статус
// и при необходимости выполняет инициализацию, переиспользуя логику Check.
func (c *Client) Init(ctx context.Context) error {
	// 1. Делаем предварительную проверку статуса.
	// Метод Check сам разберется: сканировать сеть или брать жесткий Host,
	// очистит префиксы, сделает GET-запрос и вернет строковый статус.
	status, err := c.Check(ctx)
	if err != nil {
		return fmt.Errorf("предварительная проверка статуса перед инициализацией провалена: %w", err)
	}

	fmt.Printf("Текущий статус сервера LMCZ: %s\n", status)

	// 2. Бизнес-логика: отправляем токен только если сервер не настроен
	if status != "not_configured" {
		fmt.Println("Сервер LMCZ уже настроен. Отправка токена инициализации не требуется.")
		return nil
	}

	fmt.Println("Сервер не настроен. Запуск отправки токена...")

	// 3. Вызываем приватный метод для отправки токена.
	// Так как в Check мы уже гарантированно проверили адрес, мы можем вызвать
	// вспомогательную функцию для получения правильного hostPort, чтобы не сканировать сеть дважды.
	hostPort := c.getTargetHostPort()
	if err := c.sendToken(ctx, hostPort); err != nil {
		return fmt.Errorf("ошибка инициализации сервера: %w", err)
	}

	return nil
}

// sendToken — приватный метод (с маленькой буквы). Отвечает строго за техническую
// отправку POST-запроса на конкретный хост. Извне пакета его вызвать нельзя.
func (c *Client) sendToken(ctx context.Context, hostPort string) error {
	payload := RequestPayload{Token: c.cfg.TokenXAPIKEY}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	// Переименовали переменную в targetURL, чтобы избежать затенения пакета "net/url"
	targetURL := fmt.Sprintf("http://%s/api/v2/init", hostPort)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.cfg.User, c.cfg.Pass)

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
		var errResult ResponsePayload

		// Пытаемся прочитать JSON-ответ от сервера, чтобы узнать точную причину ошибки
		if err := json.NewDecoder(resp.Body).Decode(&errResult); err == nil && errResult.Message != "" {
			return fmt.Errorf("сервер вернул ошибку (%s): %s", resp.Status, errResult.Message)
		}

		// Если сервер прислал не JSON (например, ошибку nginx), отдаем стандартный статус
		return fmt.Errorf("unexpected status code: %d (%s)", resp.StatusCode, resp.Status)
	}

	// Если статус 2xx — успешно декодируем финальный ответ
	var result ResponsePayload
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode JSON response: %w", err)
	}

	fmt.Printf("Ответ сервера на инициализацию: %s (Сообщение: %s)\n", result.Status, result.Message)
	return nil
}

// Вспомогательный приватный метод, чтобы не дублировать логику склейки хоста и порта
func (c *Client) getTargetHostPort() string {
	if c.cfg.Host == "" {
		// Если хост пустой, значит на этапе Check мы сканировали сеть.
		// Чтобы не запускать ScanSubnet повторно (это долго), сканер должен уметь
		// возвращать кешированный или первый найденный результат, либо
		// (если ScanSubnet быстрый) вызываем его:
		return scanner.ScanSubnet(c.cfg.Subnet, c.cfg.Port, c.cfg.ScannerTimeout)
	}

	cleanHost := strings.TrimPrefix(c.cfg.Host, "http://")
	cleanHost = strings.TrimPrefix(cleanHost, "https://")
	return net.JoinHostPort(cleanHost, c.cfg.Port)
}
