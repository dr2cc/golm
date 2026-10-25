package lmcz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/dr2cc/golm/internal/scanner"
	"github.com/dr2cc/golm/internal/util/urlutil"
)

// [GUI-рефакторинг] Раньше Status сам и искал адрес сервера, и делал запрос.
// Из-за этого Init (которому адрес нужен еще раз для отправки токена) был вынужден
// повторно сканировать подсеть. Теперь Status — это две независимые части:
// resolveHostPort (найти адрес) + statusAt (спросить статус по готовому адресу).
// Для внешнего кода (CLI, GUI) Status ведет себя точно так же, как раньше.
func (c *CzClient) Status(ctx context.Context) (CheckResult, error) {
	hostPort, err := c.resolveHostPort()
	if err != nil {
		return CheckResult{}, err
	}
	return c.statusAt(ctx, hostPort)
}

// [GUI-рефакторинг] resolveHostPort — бывшее начало метода Status, вынесено без изменений логики.
// Возвращает "host:port" сервера ЛМ ЧЗ: либо из конфига, либо найденный сканером подсети.
func (c *CzClient) resolveHostPort() (string, error) {
	var hostPort string

	if c.cfg.Host == "" {
		// Находим компьютеры с портом сервиса ЛМ ЧЗ и получаем первый результат
		// [GUI-рефакторинг] Сканер теперь пишет в c.out, а не всегда в консоль.
		hostPort = scanner.ScanSubnetTo(c.out, c.cfg.Subnet, c.cfg.Port, c.cfg.ScannerTimeout)
	} else {
		// Очищаем хост от схемы
		hostPort = urlutil.JoinHostPort(c.cfg.Host, c.cfg.Port)
	}

	if hostPort == "" {
		return "", fmt.Errorf("хостов с открытым портом %s не найдено", c.cfg.Port)
	}
	return hostPort, nil
}

// [GUI-рефакторинг] statusAt — бывшее продолжение метода Status: запрос GET /status
// по уже известному адресу. Код ниже не менялся, кроме замены fmt.Printf на fmt.Fprintf(c.out, ...).
func (c *CzClient) statusAt(ctx context.Context, hostPort string) (CheckResult, error) {
	// Добавляем схему
	finalURL := fmt.Sprintf("http://%s/api/v2/status", hostPort)

	fmt.Fprintf(c.out, "Проверяем статус ЛМ ЧЗ по адресу: %s\n", finalURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, finalURL, nil)
	if err != nil {
		return CheckResult{}, fmt.Errorf("creating request: %w", err)
	}

	// Формат ответа в рамках HTTP это *Response
	httpResponse, err := c.httpClient.Do(req)
	if err != nil {
		// Распаковываем ошибку http-клиента
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return CheckResult{}, fmt.Errorf("сервер LMCZ недоступен по адресу %s: %w", hostPort, urlErr.Err)
		}
		return CheckResult{}, fmt.Errorf("запрос к LMCZ завершился ошибкой: %w", err)
	}
	// ЗОЛОТОЕ ПРАВИЛО Go для HTTP-запросов:
	// МОЖНО обращаться к resp.Body (ниже) ТОЛЬКО в том случае, если err == nil.

	// КАЖДЫЙ РАЗ, когда успешно получен http.Response (err == nil и resp != nil),
	// нужно вызвать resp.Body.Close() (регистрируем закрытие тела)
	defer httpResponse.Body.Close()

	// Проверяем статус-код ответа (Чистый код: сервер ответил, но статус плохой)
	if httpResponse.StatusCode != http.StatusOK {
		return CheckResult{Status: "", HttpCode: httpResponse.StatusCode}, fmt.Errorf("сервер LMCZ вернул статус: %s", httpResponse.Status)
	}

	// 📌 Считываем содержимое тела. Три популярных метода:

	// 1. Получаем всё как строку или байты (маленькие ответы, JSON, HTML)
	b, err := io.ReadAll(httpResponse.Body)
	// Так как "свиток" (scroll) resp.Body это однонаправленный поток (stream), не возможно «перемотать» его назад.
	// Как только данные будут прочитаны (например, с помощью io.ReadAll(resp.Body)), повторное чтение вернет io.EOF (конец файла).
	if err != nil {
		return CheckResult{Status: "", HttpCode: httpResponse.StatusCode}, fmt.Errorf("failed to read Response.Body: %w", err)
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

	// Коротко про "стандартный поток вывода" (stdout).
	// В операционных системах (Linux, Windows, macOS)
	// у каждой программы при старте есть три стандартных потока данных:
	// - Стандартный ввод (stdin) — то, что пользователь вводит с клавиатуры в консоль.
	// - Стандартный вывод (stdout) — то, куда программа выводит свой обычный текст.
	// fmt.Printf и fmt.Println по умолчанию пишут именно сюда. Вы видите этот текст прямо в терминале.
	// - Стандартный вывод ошибок (stderr) — специальный отдельный поток для ошибок. Туда пишет, например, log.Println.

	// Парсим поле status из ответа
	var check StatusEndpointResponse
	if err := json.Unmarshal(b, &check); err != nil {
		// Если сервер вернул не JSON, но ответил 200 OK,
		// возвращаем сырой текст (на всякий случай)
		return CheckResult{string(b), httpResponse.StatusCode}, nil
	}

	return CheckResult{check.Status, httpResponse.StatusCode}, nil
}
