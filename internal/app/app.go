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

	// // Лядский llm!
	// // Он тут "создает абсолютно пустой («сырой») клиент, у которого поле Transport равно nil"
	// // Откуда хренова туча проблем и не контроллируемое поведение!
	//////////////////////////////////////////////// Переписать как в cli
	// 1. Создаем общий HTTP-клиент без жесткого внутреннего таймаута,
	// так как время контролируется через ctx сверху
	httpClient := &http.Client{}

	// 2. Initialize domain clients

	// Логика dm
	if cfg.LaunchDataMobile {
		log.Println("Запуск проверки DataMobile-Apache-1C...")
		// Собираем конфигурацию специально под нужды этого клиента
		clientCfg := datamobile.ClientConfig{
			Apache:     cfg.Apache,
			DataMobile: cfg.DataMobile,
		}
		dmClient := datamobile.NewDmClient(httpClient, clientCfg)

		return dmClient.Check(ctx)
	}

	// Логика lm
	// Эндпойнт /status
	if cfg.LaunchLmczStatus {
		log.Println("Запуск проверки ЛМ ЧЗ...")
		czClient := lmcz.NewCzClient(cfg.LMCZ, httpClient)
		checkResult, err := czClient.Status(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Проверка завершена успешно, (%d) status: %s\n", checkResult.HttpCode, checkResult.Status)
		return nil
	}

	// Эндпойнт /init
	if cfg.LaunchLmczInit {
		log.Printf("Отправка токена для инициализации %s:%s...\n", cfg.LMCZ.Host, cfg.LMCZ.Port)
		czClient := lmcz.NewCzClient(cfg.LMCZ, httpClient)

		return czClient.Init(ctx)
	}

	exeName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Hи один клиент не был выбран через флаги запуска.\nИспользуйте \"%s <команда> -h\" для просмотра флагов конкретной команды.\n", exeName)
	return nil
}
