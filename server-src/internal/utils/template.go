package utils

import (
	"embed"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func LoadTemplates(pattern string) *template.Template {
	tmpl := template.New("").Funcs(TemplateFuncMap())

	root := findResourceDir(pattern)

	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Printf("模板扫描失败: %v", err)
		return tmpl
	}

	for _, file := range files {
		name := strings.TrimPrefix(file, root+string(os.PathSeparator))
		name = filepath.ToSlash(name)

		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("读取模板失败 %s: %v", file, err)
			continue
		}

		t := tmpl.New(name)
		if _, err := t.Parse(string(content)); err != nil {
			log.Printf("解析模板失败 %s: %v", file, err)
		}
	}

	log.Printf("已加载 %d 个模板文件 (目录: %s)", len(files), root)
	return tmpl
}

func LoadTemplatesFromFS(efs embed.FS, baseDir string) *template.Template {
	tmpl := template.New("").Funcs(TemplateFuncMap())

	var count int
	entries, err := efs.ReadDir(baseDir)
	if err != nil {
		log.Printf("嵌入模板根目录读取失败: %v", err)
		return tmpl
	}

	var walk func(dir string)
	walk = func(dir string) {
		entries, readErr := efs.ReadDir(dir)
		if readErr != nil {
			return
		}
		for _, entry := range entries {
			fullPath := dir + "/" + entry.Name()
			if entry.IsDir() {
				walk(fullPath)
			} else if strings.HasSuffix(entry.Name(), ".html") {
				name := strings.TrimPrefix(fullPath, baseDir+"/")
				content, readErr := efs.ReadFile(fullPath)
				if readErr != nil {
					log.Printf("读取嵌入模板失败 %s: %v", fullPath, readErr)
					continue
				}
				t := tmpl.New(name)
				if _, parseErr := t.Parse(string(content)); parseErr != nil {
					log.Printf("解析嵌入模板失败 %s: %v", fullPath, parseErr)
					continue
				}
				count++
			}
		}
	}

	for _, entry := range entries {
		fullPath := baseDir + "/" + entry.Name()
		if entry.IsDir() {
			walk(fullPath)
		} else if strings.HasSuffix(entry.Name(), ".html") {
			name := strings.TrimPrefix(fullPath, baseDir+"/")
			content, readErr := efs.ReadFile(fullPath)
			if readErr != nil {
				log.Printf("读取嵌入模板失败 %s: %v", fullPath, readErr)
				continue
			}
			t := tmpl.New(name)
			if _, parseErr := t.Parse(string(content)); parseErr != nil {
				log.Printf("解析嵌入模板失败 %s: %v", fullPath, parseErr)
				continue
			}
			count++
		}
	}

	log.Printf("已加载 %d 个嵌入模板文件 (目录: %s)", count, baseDir)
	return tmpl
}

func findResourceDir(subDir string) string {
	if _, err := os.Stat(subDir); err == nil {
		abs, _ := filepath.Abs(subDir)
		return abs
	}

	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, subDir)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	cwd, _ := os.Getwd()
	return filepath.Join(cwd, subDir)
}
