Разработай подробную архитектуру модуля entity/filer который реализует следующий функционал:
- работу с файловой системой, при этом файлы могут размещаться как на диске так и в файловой системе в памяти.
- при разработке методов работы с файловой системой в памяти используй расположенные в конце файла материал **Варианты реализации и важные моменты**
- модуль должен реализовывать следующие файловые операции:
- MkdirTemp
- MkdirAll
- RemoveAll
- Create
- Stat
- IsNotExist
- Remove
- CreateTemp
- WriteFile
- ReadDir
- Rename
- Open
- Getwd
- Chdir
- ReadFile
- WriteFile
- OpenFile
- Chmod
- Chown
- при создании экземпляра модуля можно указать путь к файловой системе на диске.
- если путь к файловой системе на диске не указан, то файловая система будет создана каталоге os.TempDir() + filer.DefaultDir.
- если при создании передается параметр MemoryFS, то файловая система будет создана в памяти.
- если при создании передается параметр DiskFS, то файловая система будет создана на диске.

Не приступай к реализации модуля, только разработай архитектуру.
Запиши разработанную архитектуру в файл entity/filer/arch.md.

**Варианты реализации и важные моменты**
```go
import (
    "runtime"
    "os"
    "path/filepath"
)

func GetTempDir() string {
    if runtime.GOOS == "linux" {
        // Использовать /dev/shm (RAM-диск в Linux)
        if _, err := os.Stat("/dev/shm"); err == nil {
            return "/dev/shm/myapp"
        }
    }
    return os.TempDir()
}
```
### 1. **Создание директории перед использованием**
```go
func GetTempDir() (string, error) {
    var tempDir string
    
    if runtime.GOOS == "linux" {
        if _, err := os.Stat("/dev/shm"); err == nil {
            tempDir = "/dev/shm/myapp"
        }
    }
    
    if tempDir == "" {
        tempDir = filepath.Join(os.TempDir(), "myapp")
    }
    
    // ВАЖНО: создать директорию, если её нет
    if err := os.MkdirAll(tempDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create temp dir: %w", err)
    }
    
    return tempDir, nil
}
```

### 2. **Пример использования с внешними программами**
```go
import (
    "os/exec"
    "path/filepath"
)

func RunExternalCommand() error {
    tempDir, err := GetTempDir()
    if err != nil {
        return err
    }
    
    // Создаём уникальную поддиректорию для операции
    workDir := filepath.Join(tempDir, fmt.Sprintf("work_%d", time.Now().UnixNano()))
    if err := os.MkdirAll(workDir, 0755); err != nil {
        return err
    }
    defer os.RemoveAll(workDir) // Очистка после работы
    
    // Клонируем репозиторий
    cmd := exec.Command("git", "clone", "--depth=1", "https://github.com/repo.git", workDir)
    if err := cmd.Run(); err != nil {
        return err
    }
    
    // Запускаем внешнюю программу с этим путём
    cmd = exec.Command("external-tool", "--input", workDir, "--output", filepath.Join(workDir, "result"))
    cmd.Dir = workDir // Устанавливаем рабочую директорию
    
    return cmd.Run()
}
```

## ⚠️ Важные ограничения /dev/shm:

### 1. **Ограничение по размеру**
```go
import "github.com/shirou/gopsutil/v3/mem"

func CheckAvailableRAM() (uint64, error) {
    v, err := mem.VirtualMemory()
    if err != nil {
        return 0, err
    }
    
    // /dev/shm обычно ограничен 50% от RAM
    shmMax := v.Total / 2
    
    // Проверяем реальный размер
    var stat syscall.Statfs_t
    if err := syscall.Statfs("/dev/shm", &stat); err == nil {
        available := stat.Bavail * uint64(stat.Bsize)
        return available, nil
    }
    
    return shmMax, nil
}
```

### 2. **Fallback механизм при нехватке места**
```go
type TempDirManager struct {
    preferRAM bool
    fallbackDir string
}

func (m *TempDirManager) GetWorkDir(estimatedSize int64) (string, error) {
    if m.preferRAM && runtime.GOOS == "linux" {
        available, _ := CheckAvailableRAM()
        
        // Если места достаточно (с запасом 20%)
        if available > uint64(estimatedSize)*120/100 {
            return "/dev/shm/myapp", nil
        }
    }
    
    // Fallback на обычный диск
    return filepath.Join(os.TempDir(), "myapp"), nil
}
```

## 🔧 Рекомендации для работы с внешними программами:

### 1. **Изоляция процессов**
```go
func RunIsolated(workDir string, command string, args ...string) error {
    // Создаём уникальную директорию для каждого процесса
    processDir := filepath.Join(workDir, fmt.Sprintf("proc_%d_%d", 
        os.Getpid(), time.Now().UnixNano()))
    
    if err := os.MkdirAll(processDir, 0755); err != nil {
        return err
    }
    defer os.RemoveAll(processDir)
    
    cmd := exec.Command(command, args...)
    cmd.Dir = processDir
    
    // Устанавливаем переменные окружения
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("TMPDIR=%s", processDir),
        fmt.Sprintf("TEMP=%s", processDir),
        fmt.Sprintf("TMP=%s", processDir),
    )
    
    return cmd.Run()
}
```

