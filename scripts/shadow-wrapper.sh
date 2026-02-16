#!/bin/bash

# Wrapper script for shadow analyzer to output JSON format compatible with golangci-lint
# ./shadow-wrapper.sh $(find . -name '*.go' -not -path './vendor/*')

if [ $# -eq 0 ]; then
    echo "Usage: $0 <go-files>"
    exit 1
fi

# Run shadow analyzer and capture output
shadow_output=$(shadow "$@" 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ]; then
    echo '{"Issues": []}'
    exit 0
fi

# Parse shadow output and convert to JSON format
echo '{"Issues": ['
first=true
while IFS= read -r line; do
    if [[ $line =~ ^(.+):([0-9]+):([0-9]+):[[:space:]]*(.*)$ ]]; then
        file="${BASH_REMATCH[1]}"
        line_num="${BASH_REMATCH[2]}"
        col_num="${BASH_REMATCH[3]}"
        message="${BASH_REMATCH[4]}"
        
        # Skip files in vendor directory
        if [[ $file != vendor/* && $file != */vendor/* ]]; then
            if [ "$first" = true ]; then
                first=false
            else
                echo ','
            fi
            
            echo "{"
            echo "  \"resource\": \"$file\","
            echo "  \"owner\": \"_generated_diagnostic_collection_name_#2\","
            echo "  \"code\": {"
            echo "    \"value\": \"default\","
            echo "    \"target\": {"
            echo "      \"\$mid\": 1,"
            echo "      \"path\": \"/golang.org/x/tools/go/analysis/passes/shadow\","
            echo "      \"scheme\": \"https\","
            echo "      \"authority\": \"pkg.go.dev\""
            echo "    }"
            echo "  },"
            echo "  \"severity\": 4,"
            echo "  \"message\": \"$message\","
            echo "  \"source\": \"shadow\","
            echo "  \"startLineNumber\": $line_num,"
            echo "  \"startColumn\": $col_num,"
            echo "  \"endLineNumber\": $line_num,"
            echo "  \"endColumn\": $((col_num + 3)),"
            echo "  \"extensionID\": \"golang.go\""
            echo "}"
        fi
    fi
done <<< "$shadow_output"
echo ']}'