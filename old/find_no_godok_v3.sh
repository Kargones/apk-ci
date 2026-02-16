#!/bin/bash

# Поиск всех .go файлов исключая _test.go в каталоге internal
find internal -name "*.go" ! -name "*_test.go" -type f -print0 | while IFS= read -r -d '' file; do
#   echo "Файл: $file"
    # Получаем номера строк с функциями
    grep -n "^[[:space:]]*func[[:space:]]\+\([^(]*[[:space:]]\+\)\?[A-Z][a-zA-Z0-9_]*[[:space:]]*(" "$file" | cut -d: -f1 | while read -r line_num; do
    # echo "line_num: $line_num"
        # Получаем 3 строки перед функцией
        start_line=$((line_num > 4  ? line_num - 4 : 1))
        end_line=$((line_num - 1))
        
        # Считаем комментарии в этих строках
        comment_count=$(sed -n "${start_line},${end_line}p" "$file" | grep -c "^//")
        
        if [ "$comment_count" -lt 2 ]; then
            func_line=$(sed -n "${line_num}p" "$file")
            echo "Файл: $file"
            echo "Строка $line_num: $func_line"
            echo "---"
        fi
    done
done