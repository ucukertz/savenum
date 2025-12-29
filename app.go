package main

import (
    "context"
    "fmt"
    "io"
    "math"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "bytes"

    "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
    ctx context.Context
}

func NewApp() *App {
    return &App{}
}

func (a *App) startup(ctx context.Context) {
    a.ctx = ctx
}

func (a *App) SelectDirectory() (string, error) {
    return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
        Title: "Select Destination Directory",
    })
}

func (a *App) SaveFile(destDir, baseName string, digits int, fileName string, fileData []byte, sourceURL string) (string, error) {
    
    ext := filepath.Ext(fileName)
    if ext == "" {
        ext = ".jpg" 
    }

    var reader io.Reader
    var contentLength int64

    if sourceURL != "" {
        resp, err := http.Get(sourceURL)
        if err != nil {
            return "", fmt.Errorf("failed to download file: %v", err)
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != 200 {
            return "", fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
        }

        if ext == "" || ext == ".html" || ext == ".php" {
            switch resp.Header.Get("Content-Type") {
            case "image/png":
                ext = ".png"
            case "image/jpeg":
                ext = ".jpg"
            case "image/webp":
                ext = ".webp"
            case "image/gif":
                ext = ".gif"
            }
        }

        reader = resp.Body
        contentLength = resp.ContentLength
    } else {
        reader = bytes.NewReader(fileData)
        contentLength = int64(len(fileData))
    }

    usedNumbers := make(map[int]bool)
    
    files, err := os.ReadDir(destDir)
    if err != nil {
        return "", fmt.Errorf("failed to read destination directory: %v", err)
    }

    patternStr := fmt.Sprintf(`^%s(\d{%d})%s$`, regexp.QuoteMeta(baseName), digits, regexp.QuoteMeta(ext))
    re := regexp.MustCompile(patternStr)

    for _, f := range files {
        matches := re.FindStringSubmatch(f.Name())
        if len(matches) > 1 {
            num, err := strconv.Atoi(matches[1])
            if err == nil {
                usedNumbers[num] = true
            }
        }
    }

    nextNum := 1
    for {
        if !usedNumbers[nextNum] {
            break
        }
        nextNum++
    }

    limit := int(math.Pow(10, float64(digits)))
    if nextNum >= limit {
        return "", fmt.Errorf("digit overflow: cannot fit number %d into %d digits", nextNum, digits)
    }

    formatStr := "%0" + strconv.Itoa(digits) + "d"
    numberSuffix := fmt.Sprintf(formatStr, nextNum)
    newFileName := fmt.Sprintf("%s%s%s", baseName, numberSuffix, ext)
    destPath := filepath.Join(destDir, newFileName)

    dstFile, err := os.Create(destPath)
    if err != nil {
        return "", fmt.Errorf("failed to create file: %v", err)
    }
    defer dstFile.Close()

    written, err := io.Copy(dstFile, reader)
    if err != nil {
        os.Remove(destPath)
        return "", fmt.Errorf("failed to write file: %v", err)
    }

    if contentLength > 0 && written != contentLength {
        os.Remove(destPath)
        return "", fmt.Errorf("file size mismatch")
    }

    return newFileName, nil
}