package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dr2cc/golm/internal/util/urlutil"
	"go.yaml.in/yaml/v3"
)

type ApacheConfig struct {
	PublicationName string        `yaml:"publication_name"`
	Address         string        `yaml:"address"`
	Port            string        `yaml:"port"`
	ConfPath        string        `yaml:"conf_path"`
	WarningDuration time.Duration `yaml:"warning_duration"`
}

type DataMobileConfig struct {
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type LmczConfig struct {
	Host           string        `yaml:"host"`
	Subnet         string        `yaml:"subnet"`
	Port           string        `yaml:"port"`
	ScannerTimeout time.Duration `yaml:"scanner_timeout"`
	User           string        `yaml:"user"`
	Pass           string        `yaml:"pass"`
	TokenXAPIKEY   string        `yaml:"token_x_api_key"`
}

type Config struct {
	Env                  string           `yaml:"env"`
	ClientRequestTimeout time.Duration    `yaml:"client_request_timeout"`
	LaunchDataMobile     bool             `yaml:"launch_datamobile"`
	LaunchLmczStatus     bool             `yaml:"launch_lmcz_check"`
	LaunchLmczInit       bool             `yaml:"launch_lmcz_init"`
	Apache               ApacheConfig     `yaml:"apache"`
	DataMobile           DataMobileConfig `yaml:"datamobile"`
	LMCZ                 LmczConfig       `yaml:"lmcz"`
}

// Рефакторинг добавления port в конфигурацию.
// BaseURL собирает полный префикс адреса (включая протокол и порт) для HTTP-запросов.
// Метод не нужно прописывать в YAML, он работает на основе существующих полей.
func (a ApacheConfig) BaseURL() string {
	// Ваша функция уберет http:// и красиво склеит хост с портом
	cleanHostPort := urlutil.JoinHostPort(a.Address, a.Port)

	// Возвращаем готовую строку, которую сразу поймет http.Client
	return "http://" + cleanHostPort
}

// [GUI-рефакторинг] DefaultFileName — имя файла конфигурации, которое раньше
// было зашито строкой внутри New(). Вынесено в константу, чтобы GUI использовал то же имя.
const DefaultFileName = "config.yaml"

// New() парсит флаги и возвращает готовую конфигурацию.
// Когда будем читать переменные окружения или .env файл, код поменяется только тут.
//
// [GUI-рефакторинг] Раньше New() одновременно читал YAML и вызывал flag.Parse().
// GUI флаги командной строки не нужны (а повторный flag.Parse() / flag.BoolVar
// с теми же именами вызвал бы панику), поэтому чтение файла вынесено в Load().
// Для CLI ничего не изменилось: New() = Load("config.yaml") + флаги + проверка среды.
func New() (*Config, error) {
	// 1-3. Читаем и парсим файл конфигурации (теперь это делает Load)
	cfg, err := Load(DefaultFileName)
	if err != nil {
		return nil, err
	}

	// 4. Привязываем флаги командной строки к полям созданного объекта.
	// В качестве дефолтных значений передаем то, что прочитано из файла YAML.
	flag.BoolVar(&cfg.LaunchDataMobile, "dm", cfg.LaunchDataMobile, "Запустить проверку DataMobile-Apache-1C")
	flag.BoolVar(&cfg.LaunchLmczStatus, "status", cfg.LaunchLmczStatus, "Запустить тест ЛМ ЧЗ")
	flag.BoolVar(&cfg.LaunchLmczInit, "token", cfg.LaunchLmczInit, "Инициализировать ЛМ ЧЗ")

	// Переопределяем стандартный вывод ошибок и подсказок, чтобы не получить странную строку при неправильном флаге.
	flag.Usage = func() {
		// Подменяем os.Args[0] на красивое имя
		os.Args[0] = "golm"

		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	// 5. Парсим флаги.
	// Если пользователь передаст флаг в терминале (-dm),
	// он перепишет значение, которое было в config.yaml
	flag.Parse()

	// Вызываем проверку проблем среды уже при создании конфига
	cfg.validateEnvironment()

	return cfg, nil
}

// [GUI-рефакторинг] Load — бывшие шаги 1-3 из New(): только чтение и парсинг YAML,
// без флагов командной строки и без печати предупреждений. Используется и CLI (через New),
// и GUI (напрямую, с путем к config.yaml рядом с exe).
func Load(filename string) (*Config, error) {
	// 1. Инициализируем пустую структуру
	cfg := &Config{}

	// 2. Читаем файл конфигурации
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации %s: %w", filename, err)
	}

	// 3. Парсим YAML в структуру.
	// Библиотека yaml.v3 превращает строки вида "10s" или "5m" в тип time.Duration
	if err := yaml.Unmarshal(fileBytes, cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга YAML: %w", err)
	}

	return cfg, nil
}

// [GUI-рефакторинг] Save записывает конфигурацию в файл (нужно для вкладки «Настройки» в GUI).
// Простой yaml.Marshal(cfg) стер бы все комментарии в config.yaml (подсказки вида
// "# 192.168.0.75 / localhost"), поэтому делаем иначе:
//  1. читаем существующий файл как дерево yaml.Node — в нем хранятся и комментарии;
//  2. кодируем новую конфигурацию в такое же дерево;
//  3. переносим новые значения в старое дерево (mergeNodes), комментарии остаются на месте.
//
// Если файла еще нет — просто записываем новую конфигурацию.
func Save(filename string, cfg *Config) error {
	var fresh yaml.Node
	if err := fresh.Encode(cfg); err != nil {
		return fmt.Errorf("ошибка сериализации конфигурации: %w", err)
	}

	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{&fresh}}

	if oldBytes, err := os.ReadFile(filename); err == nil {
		var old yaml.Node
		if err := yaml.Unmarshal(oldBytes, &old); err == nil && old.Kind == yaml.DocumentNode && len(old.Content) == 1 {
			mergeNodes(old.Content[0], &fresh)
			doc = &old
		}
	}

	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("ошибка сериализации конфигурации: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("ошибка сериализации конфигурации: %w", err)
	}

	// Пишем сначала во временный файл и только потом подменяем исходный:
	// если запись оборвется на середине, config.yaml останется целым.
	tmp := filename + ".tmp"
	if err := os.WriteFile(tmp, []byte(buf.String()), 0o600); err != nil {
		return fmt.Errorf("не удалось записать файл конфигурации %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, filename); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("не удалось сохранить файл конфигурации %s: %w", filename, err)
	}
	return nil
}

// [GUI-рефакторинг] mergeNodes переносит значения из src в dst (оба — YAML-словари),
// не трогая комментарии и порядок ключей в dst. Ключи, которых в dst нет, добавляются в конец.
func mergeNodes(dst, src *yaml.Node) {
	if dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		return
	}
	// В MappingNode.Content ключи и значения идут парами: [ключ1, значение1, ключ2, значение2, ...]
	for i := 0; i+1 < len(src.Content); i += 2 {
		key, val := src.Content[i], src.Content[i+1]

		var target *yaml.Node
		for j := 0; j+1 < len(dst.Content); j += 2 {
			if dst.Content[j].Value == key.Value {
				target = dst.Content[j+1]
				break
			}
		}

		switch {
		case target == nil:
			dst.Content = append(dst.Content, key, val)
		case target.Kind == yaml.MappingNode && val.Kind == yaml.MappingNode:
			mergeNodes(target, val)
		case target.Kind == yaml.ScalarNode && val.Kind == yaml.ScalarNode:
			// Меняем только значение и тип; стиль (кавычки) и комментарии остаются прежними
			target.Value = val.Value
			target.Tag = val.Tag
		default:
			// Структура значения изменилась (например, было число, стал словарь) — заменяем целиком
			*target = *val
		}
	}
}

// Внутренний метод для проверки потенциальных проблем среды
func (c *Config) validateEnvironment() {
	if !isWSL() {
		return
	}

	// Если мы в WSL и адрес локальный — выводим предупреждение
	if strings.Contains(c.Apache.Address, "127.0.0.1") || strings.Contains(c.Apache.Address, "localhost") {
		fmt.Println("[CONFIG WARNING]: Вы запускаете код внутри WSL2 и запрашиваете localhost (127.0.0.1).")
		fmt.Println("Если сетевой режим WSL не изменен на 'mirrored', запрос завершится ошибкой 'connection refused'.")
	}
}

// Сама функция проверки среды (не экспортируется наружу, так как нужна только здесь)
func isWSL() bool {
	version, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(version))
	return strings.Contains(content, "microsoft") || strings.Contains(content, "wsl")
}
