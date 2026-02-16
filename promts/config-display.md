Используй код расположенный ниже как основу для создания функуии displayConfig
Замени этой функцией функцию checkXvfb

pkill -f Xvfb || true

# Очистка lock файлов
echo "Очистка lock файлов..."
rm -f /tmp/.X99-lock /tmp/.X*-lock 2>/dev/null || true

# Запуск виртуального дисплея
echo "Запуск виртуального дисплея :99..."
Xvfb :99 -screen 0 1920x1080x24 -ac +extension GLX +render -noreset > /dev/null 2>&1 &
XVFB_PID=$!

# Ожидание запуска Xvfb
sleep 3

# Проверка запуска Xvfb
if ps -p $XVFB_PID > /dev/null; then
    echo "✓ Xvfb успешно запущен (PID: $XVFB_PID)"
else
    echo "✗ Ошибка запуска Xvfb"
    exit 1
fi

# Установка переменных окружения
echo "Установка переменных окружения..."
export DISPLAY=:99
export XAUTHORITY=/tmp/.Xauth99

# Создание файла авторизации X11
echo "Создание файла авторизации X11..."
touch $XAUTHORITY
xauth add :99 . $(xxd -l 16 -p /dev/urandom) 2>/dev/null || true

# Проверка подключения к дисплею
echo "Проверка подключения к дисплею..."
if xdpyinfo -display :99 > /dev/null 2>&1; then
    echo "✓ Подключение к дисплею :99 успешно"
else
    echo "✗ Не удается подключиться к дисплею :99"
    echo "Попытка исправления..."
    
    # Альтернативный запуск с другими параметрами
    pkill -f Xvfb || true
    sleep 2
    Xvfb :99 -screen 0 1920x1080x24 -dpi 96 -ac +extension RANDR > /dev/null 2>&1 &
    sleep 3
    
    if xdpyinfo -display :99 > /dev/null 2>&1; then
        echo "✓ Подключение к дисплею :99 успешно (после исправления)"
    else
        echo "✗ Все еще не удается подключиться к дисплею"
        exit 1
    fi
fi

# Добавление переменных в bashrc для постоянного использования
echo "Добавление переменных в ~/.bashrc..."
echo "" >> ~/.bashrc
echo "# Переменные для работы с виртуальным дисплеем" >> ~/.bashrc
echo "export DISPLAY=:99" >> ~/.bashrc
echo "export XAUTHORITY=/tmp/.Xauth99" >> ~/.bashrc

# Создание скрипта для быстрого перезапуска дисплея
cat > /usr/local/bin/restart-display << 'EOF'
#!/bin/bash
echo "Перезапуск виртуального дисплея..."
pkill -f Xvfb || true
sleep 2
rm -f /tmp/.X99-lock /tmp/.X*-lock 2>/dev/null || true
Xvfb :99 -screen 0 1920x1080x24 -ac +extension GLX +render -noreset > /dev/null 2>&1 &
sleep 3
export DISPLAY=:99
export XAUTHORITY=/tmp/.Xauth99
echo "Дисплей перезапущен. DISPLAY=$DISPLAY"
EOF

chmod +x /usr/local/bin/restart-display