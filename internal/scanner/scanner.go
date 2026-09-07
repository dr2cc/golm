package scanner

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ScanSubnet сканирует сеть и возвращает первый ответивший IP.
// При этом сканирование всей сети всё равно гарантированно дорабатывает до конца.
func ScanSubnet(subnet, port string, timeout time.Duration) string {
	var wg sync.WaitGroup

	fmt.Printf("Запуск сканирования подсети %s.0/24 по порту %s...\n", subnet, port)

	sem := make(chan struct{}, 50)

	// Канал с буфером на 254 элемента, чтобы горутины не блокировались
	// при отправке, даже если основная функция уже перестала читать из канала
	foundChan := make(chan string, 254)

	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("%s.%d", subnet, i)
		address := net.JoinHostPort(ip, port)

		wg.Add(1)
		sem <- struct{}{}

		go func(addr string) {
			defer wg.Done()
			defer func() { <-sem }()

			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err != nil {
				return
			}
			conn.Close()

			fmt.Printf("[+] Открыт: %s\n", addr)
			foundChan <- addr // Отправляем адрес в канал
		}(address)
	}

	// Запускаем ожидание завершения всех горутин в отдельном потоке.
	// Как только все закончат работу — закрываем канал.
	go func() {
		wg.Wait()
		close(foundChan)
	}()

	// Читаем из канала самое первое значение.
	// Если в канал что-то пришло, мы тут же возвращаем это из функции.
	// Если сеть пуста, канал закроется в горутине выше, и из него вернется пустота.
	firstFound := <-foundChan

	return firstFound
}
