package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/datamobile"
	"github.com/dr2cc/golm/internal/lmcz"
)

func Run(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ClientRequestTimeout)
	// ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Создаем общий HTTP-клиент без жесткого внутреннего таймаута,
	// так как время контролируется через ctx сверху
	httpClient := &http.Client{}

	// 2. Initialize domain clients

	// Логика dm
	if cfg.LaunchDataMobile {
		log.Println("Запуск проверки DataMobile-Apache-1C...")
		// Собираем конфиг специально под нужды клиента
		clientCfg := datamobile.ClientConfig{
			Apache:     cfg.Apache,
			DataMobile: cfg.DataMobile,
		}
		dmClient := datamobile.NewClient(clientCfg, httpClient)

		return dmClient.Check(ctx)
	}

	// Логика lm
	// Эндпойнт /status
	if cfg.LaunchLmczStatus {
		log.Println("Запуск проверки ЛМ ЧЗ...")
		czClient := lmcz.NewClient(cfg.LMCZ, httpClient)
		checkResult, err := czClient.Status(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Проверка завершена успешно, (%d) status: %s\n", checkResult.HttpCode, checkResult.Status)
		return nil
	}

	// Эндпойнт /init
	if cfg.LaunchLmczInit {
		log.Printf("Отправляем токен для инициализации %s...\n", cfg.LMCZ.Host)
		// log.Println("Инициализация ЛМ ЧЗ...")
		czClient := lmcz.NewClient(cfg.LMCZ, httpClient)

		return czClient.Init(ctx)
	}

	exeName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Hи один клиент не был выбран через флаги запуска.\nИспользуйте \"%s <команда> -h\" для просмотра флагов конкретной команды.\n", exeName)
	// log.Println("Hи один клиент не был выбран через флаги запуска.")
	return nil
}
