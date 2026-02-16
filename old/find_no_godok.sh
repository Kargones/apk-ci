#!/bin/bash

# Поиск всех .go файлов исключая _test.go в каталоге internal
find internal -name "*.go" ! -name "*_test.go" -type f | while read file; do
    # Читаем файл построчно
    line_number=0
    while IFS= read -r line; do
        ((line_number++))
        
        # Проверяем, является ли строка объявлением функции
        if [[ "$line" =~ ^func\ [^(]*\ [A-Z][a-zA-Z0-9_]*\ *\( ]]; then
            # Извлекаем имя функции
            func_name=$(echo "$line" | grep -oP 'func [^(]* \K[A-Z][a-zA-Z0-9_]*')
            
            if [[ -n "$func_name" ]]; then
                # Считаем количество комментариев перед функцией
                comment_count=0
                prev_line=$((line_number - 1))
                
                # Проверяем предыдущие строки (максимум 10 строк назад)
                for ((i=1; i<=10 && prev_line>=1; i++)); do
                    prev_line_content=$(sed -n "${prev_line}p" "$file")
                    if [[ "$prev_line_content" =~ ^// ]]; then
                        ((comment_count++))
                    elif [[ ! "$prev_line_content" =~ ^[[:space:]]*$ ]]; then
                        # Если встретили не пустую строку и не комментарий, прерываем поиск
                        break
                    fi
                    ((prev_line--))
                done
                
                # Если комментариев меньше 2, выводим строку с функцией
                if [ "$comment_count" -lt 2 ]; then
                    echo "Файл: $file"
                    echo "Строка $line_number: $line"
                    echo "---"
                fi
            fi
        fi
    done < "$file"
done