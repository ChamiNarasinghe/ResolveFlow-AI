package main

import (
    "fmt"
    "io"
    "os"
    "strings"

    "github.com/chami2002/resolveflow-api/internal/auth"
)

func main() {
    input, err := io.ReadAll(os.Stdin)
    if err != nil {
        fmt.Fprintln(os.Stderr, "Failed to read password")
        os.Exit(1)
    }

    password := strings.TrimRight(string(input), "\r\n")

    hash, err := auth.HashPassword(password)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    fmt.Println(hash)
}
