#!/bin/bash

# Поиск всех .go файлов исключая _test.go в каталоге internal
find internal -name "*.go" ! -name "*_test.go" -type f | while read file; do
echo "Файл: $file"
    # Ищем все строки с объявлениями функций, начинающихся с большой буквы
    grep -n "func [^(]* [A-Z][a-zA-Z0-9_]* *(" "$file" | while read -r match; do
        line_num=$(echo "$match" | cut -d: -f1)
        func_line=$(echo "$match" | cut -d: -f2-)
        
        # Считаем количество комментариев перед функцией
        comment_count=0
        prev_line=$((line_num - 1))
      echo "line_num: $line_num"
        # Проверяем предыдущие строки
        while [ $prev_line -gt 0 ] && [ $prev_line -gt $((line_num - 10)) ]; do
            prev_content=$(sed -n "${prev_line}p" "$file")
            if [[ "$prev_content" =~ ^// ]]; then
                ((comment_count++))
            elif [[ ! "$prev_content" =~ ^[[:space:]]*$ ]]; then
                # Прерываем если встретили не пустую строку и не комментарий
                break
            fi
            ((prev_line--))
        done
        
        # Если комментариев меньше 2, выводим результат
        if [ "$comment_count" -lt 2 ]; then
            echo "Файл: $file"
            echo "Строка $line_num: $func_line"
            echo "Количество комментариев: $comment_count"
            echo "---"
        fi
    done
done