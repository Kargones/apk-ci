# Архитектура модуля entity/filer

## Обзор

Модуль `entity/filer` предоставляет унифицированный интерфейс для работы с файловой системой, поддерживая как дисковое хранение, так и файловую систему в памяти. Модуль обеспечивает абстракцию над различными типами файловых систем и предоставляет единый API для файловых операций.

## Основные компоненты

### 1. Интерфейсы

#### FileSystem
Основной интерфейс, определяющий контракт для всех файловых операций:

```go
type FileSystem interface {
    // Операции с директориями
    MkdirTemp(dir, pattern string) (string, error)
    MkdirAll(path string, perm os.FileMode) error
    RemoveAll(path string) error
    ReadDir(dirname string) ([]os.DirEntry, error)
    Getwd() (string, error)
    Chdir(dir string) error
    
    // Операции с файлами
    Create(name string) (File, error)
    CreateTemp(dir, pattern string) (File, error)
    Open(name string) (File, error)
    OpenFile(name string, flag int, perm os.FileMode) (File, error)
    Remove(name string) error
    Rename(oldpath, newpath string) error
    
    // Операции чтения/записи
    ReadFile(filename string) ([]byte, error)
    WriteFile(filename string, data []byte, perm os.FileMode) error
    
    // Информационные операции
    Stat(name string) (os.FileInfo, error)
    IsNotExist(err error) bool
    
    // Операции с правами доступа
    Chmod(name string, mode os.FileMode) error
    Chown(name string, uid, gid int) error
}
```

#### File
Интерфейс для работы с файловыми дескрипторами:

```go
type File interface {
    io.Reader
    io.Writer
    io.Closer
    io.Seeker
    
    Name() string
    Stat() (os.FileInfo, error)
    Sync() error
    Truncate(size int64) error
    Chmod(mode os.FileMode) error
    Chown(uid, gid int) error
}
```

### 2. Типы файловых систем

#### FSType
Перечисление типов файловых систем:

```go
type FSType int

const (
    DiskFS FSType = iota
    MemoryFS
)
```

### 3. Конфигурация

#### Config
Структура конфигурации для создания файловой системы:

```go
type Config struct {
    Type     FSType // Тип файловой системы
    BasePath string // Базовый путь для дисковой ФС
    UseRAM   bool   // Использовать RAM-диск (/dev/shm) для MemoryFS
}
```

#### Options
Функциональные опции для настройки:

```go
type Option func(*Config)

func WithDiskFS(basePath string) Option
func WithMemoryFS() Option
func WithRAMDisk() Option
```

### 4. Реализации

#### DiskFileSystem
Реализация файловой системы на диске:

```go
type DiskFileSystem struct {
    basePath string
    currentDir string
}

func NewDiskFileSystem(basePath string) (*DiskFileSystem, error)
```

**Особенности:**
- Использует стандартные операции `os` пакета
- Поддерживает все файловые операции нативно
- Автоматически создает базовую директорию при инициализации
- Если `basePath` не указан, использует `os.TempDir() + DefaultDir`

#### MemoryFileSystem
Реализация файловой системы в памяти:

```go
type MemoryFileSystem struct {
    root       *memoryNode
    currentDir string
    useRAM     bool
    ramPath    string
}

func NewMemoryFileSystem(useRAM bool) (*MemoryFileSystem, error)
```

**Особенности:**
- Использует внутреннюю структуру данных для хранения файлов и директорий
- При `useRAM=true` использует `/dev/shm` на Linux системах
- Поддерживает fallback на обычную временную директорию
- Реализует проверку доступного места в RAM

#### memoryNode
Внутренняя структура для представления узлов в памяти:

```go
type memoryNode struct {
    name     string
    isDir    bool
    mode     os.FileMode
    modTime  time.Time
    size     int64
    data     []byte
    children map[string]*memoryNode
    parent   *memoryNode
}
```

#### MemoryFile
Реализация файлового дескриптора для файловой системы в памяти:

```go
type MemoryFile struct {
    node   *memoryNode
    offset int64
    flag   int
}
```

### 5. Фабрика

#### Factory
Фабрика для создания файловых систем:

```go
type Factory struct {
    defaultConfig Config
}

func NewFactory(options ...Option) *Factory
func (f *Factory) CreateFileSystem(options ...Option) (FileSystem, error)
```

### 6. Утилиты

#### TempDirManager
Менеджер для работы с временными директориями:

```go
type TempDirManager struct {
    preferRAM   bool
    fallbackDir string
}

func (m *TempDirManager) GetWorkDir(estimatedSize int64) (string, error)
func (m *TempDirManager) CheckAvailableRAM() (uint64, error)
```

#### PathUtils
Утилиты для работы с путями:

```go
func GetOptimalTempDir() (string, error)
func EnsureDir(path string, perm os.FileMode) error
func IsRAMDiskAvailable() bool
```

## Константы

```go
const (
    DefaultDir = "filer"
    DefaultPerm = 0700
    RAMDiskPath = "/dev/shm"
    RAMDiskThreshold = 0.5 // 50% от доступной памяти
)
```

## Архитектурные решения

### 1. Паттерн Strategy
Используется для переключения между различными реализациями файловых систем через единый интерфейс `FileSystem`.

### 2. Паттерн Factory
Предоставляет удобный способ создания экземпляров файловых систем с различными конфигурациями.

### 3. Паттерн Options
Используется для гибкой настройки конфигурации через функциональные опции.

### 4. Абстракция файловых дескрипторов
Интерфейс `File` обеспечивает единообразную работу с файлами независимо от типа файловой системы.

## Особенности реализации

### 1. RAM-диск на Linux
- Автоматическое определение доступности `/dev/shm`
- Проверка доступного места перед использованием
- Fallback на обычную временную директорию при нехватке места
- Создание уникальных поддиректорий для изоляции процессов

### 2. Управление памятью
- Мониторинг использования RAM при работе с памятью
- Автоматическая очистка временных файлов
- Ограничение размера файлов в памяти

### 3. Совместимость
- Единый API для всех типов файловых систем
- Поддержка всех стандартных файловых операций
- Корректная обработка ошибок и edge cases

### 4. Безопасность
- Изоляция процессов через уникальные директории
- Корректная обработка прав доступа
- Защита от path traversal атак

## Примеры использования

### Создание дисковой файловой системы
```go
fs, err := filer.NewFactory(
    filer.WithDiskFS("/custom/path"),
).CreateFileSystem()
```

### Создание файловой системы в памяти
```go
fs, err := filer.NewFactory(
    filer.WithMemoryFS(),
    filer.WithRAMDisk(),
).CreateFileSystem()
```

### Создание с настройками по умолчанию
```go
fs, err := filer.NewFactory().CreateFileSystem()
// Использует os.TempDir() + DefaultDir
```

## Тестирование

Модуль должен включать:
- Юнит-тесты для всех интерфейсов
- Интеграционные тесты для различных сценариев
- Бенчмарки для сравнения производительности
- Тесты совместимости между различными типами ФС

## Зависимости

- `os` - стандартные файловые операции
- `io` - интерфейсы для работы с потоками
- `path/filepath` - работа с путями
- `runtime` - определение операционной системы
- `syscall` - системные вызовы для проверки `/dev/shm`
- `time` - работа с временными метками
- `fmt` - форматирование ошибок

## Расширяемость

Архитектура позволяет легко добавлять новые типы файловых систем:
- Сетевые файловые системы (NFS, SMB)
- Облачные хранилища (S3, GCS)
- Зашифрованные файловые системы
- Файловые системы с версионированием