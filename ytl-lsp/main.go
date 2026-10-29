// Отладочный запуск: показывает, как парсер видит файл.
// go run . путь\к\файлу.ytl
package main

import (
	"fmt"
	"os"
	"strings"

	"ytl-lsp/parser"
	"ytl-lsp/validate"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("укажи путь к файлу")
		return
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("не смог прочитать файл:", err)
		return
	}

	doc := parser.Parse(string(data))
	printNodes(doc.Children, 0)

	for _, e := range doc.Errors {
		fmt.Printf("ОШИБКА %d:%d  %s\n", e.Range.Start.Line+1, e.Range.Start.Col+1, e.Message)
	}
}

func printNodes(nodes []parser.Node, depth int) {
	pad := strings.Repeat("  ", depth)
	for _, n := range nodes {
		switch v := n.(type) {
		case *parser.Section:
			if v.Parent != "" {
				fmt.Printf("%s[секция] %s (from %s)\n", pad, v.Name, v.Parent)
			} else {
				fmt.Printf("%s[секция] %s\n", pad, v.Name)
			}
			printNodes(v.Children, depth+1)
		case *parser.Pair:
			if v.IsArray {
				fmt.Printf("%s%s = массив из %d элементов\n", pad, v.Key, len(v.Items))
				for _, it := range v.Items {
					fmt.Printf("%s  - %q\n", pad, it.Value)
					for _, is := range validate.Value(v.Key, it.Value) {
						fmt.Printf("%s    ⚠ строка %d: %s\n", pad, it.Range.Start.Line+1, is.Message)
					}
				}
			} else {
				fmt.Printf("%s%s = %q\n", pad, v.Key, v.Value)
				for _, is := range validate.Value(v.Key, v.Value) {
					fmt.Printf("%s  ⚠ строка %d: %s\n", pad, v.ValueRange.Start.Line+1, is.Message)
				}
			}
		}
	}
}
