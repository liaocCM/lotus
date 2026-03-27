package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func detectStacks(root string) []Stack {
	var stacks []Stack

	if s := detectGo(root); s != nil {
		stacks = append(stacks, *s)
	}
	if s := detectTypeScript(root); s != nil {
		stacks = append(stacks, *s)
	}
	if s := detectPython(root); s != nil {
		stacks = append(stacks, *s)
	}
	if s := detectRust(root); s != nil {
		stacks = append(stacks, *s)
	}
	if s := detectKotlin(root); s != nil {
		stacks = append(stacks, *s)
	}
	if s := detectSwift(root); s != nil {
		stacks = append(stacks, *s)
	}
	if s := detectDart(root); s != nil {
		stacks = append(stacks, *s)
	}

	// detect CI for all stacks
	ci := detectCI(root)
	if ci != "" && len(stacks) > 0 {
		stacks[0].CI = ci
	}

	// detect database
	db := detectDatabase(root)
	if db != "" && len(stacks) > 0 {
		stacks[0].Database = db
	}

	return stacks
}

func detectGo(root string) *Stack {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil
	}
	s := &Stack{Language: "go"}

	// extract go version
	re := regexp.MustCompile(`(?m)^go\s+(\S+)`)
	if m := re.FindSubmatch(data); len(m) > 1 {
		s.Version = string(m[1])
	}

	content := string(data)
	switch {
	case strings.Contains(content, "github.com/gin-gonic/gin"):
		s.Framework = "gin"
	case strings.Contains(content, "github.com/labstack/echo"):
		s.Framework = "echo"
	case strings.Contains(content, "github.com/gofiber/fiber"):
		s.Framework = "fiber"
	case strings.Contains(content, "github.com/gorilla/mux"):
		s.Framework = "gorilla"
	case strings.Contains(content, "connectrpc.com"):
		s.Framework = "connect"
	}

	return s
}

func detectTypeScript(root string) *Stack {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	lang := "javascript"
	if _, ok := allDeps["typescript"]; ok {
		lang = "typescript"
	}
	if _, err := os.Stat(filepath.Join(root, "tsconfig.json")); err == nil {
		lang = "typescript"
	}

	s := &Stack{Language: lang}

	switch {
	case hasKey(allDeps, "next"):
		s.Framework = "next"
	case hasKey(allDeps, "react"):
		s.Framework = "react"
	case hasKey(allDeps, "vue"):
		s.Framework = "vue"
	case hasKey(allDeps, "@angular/core"):
		s.Framework = "angular"
	case hasKey(allDeps, "svelte"):
		s.Framework = "svelte"
	case hasKey(allDeps, "express"):
		s.Framework = "express"
	case hasKey(allDeps, "@nestjs/core"):
		s.Framework = "nest"
	case hasKey(allDeps, "fastify"):
		s.Framework = "fastify"
	case hasKey(allDeps, "hono"):
		s.Framework = "hono"
	case hasKey(allDeps, "react-native"):
		s.Framework = "react-native"
	case hasKey(allDeps, "expo"):
		s.Framework = "expo"
	}

	return s
}

func detectPython(root string) *Stack {
	// check pyproject.toml, requirements.txt, setup.py
	sentinels := []string{"pyproject.toml", "requirements.txt", "setup.py", "Pipfile"}
	found := false
	for _, f := range sentinels {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	s := &Stack{Language: "python"}

	// try to detect framework from pyproject.toml or requirements.txt
	for _, f := range []string{"pyproject.toml", "requirements.txt"} {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		content := string(data)
		switch {
		case strings.Contains(content, "django"):
			s.Framework = "django"
		case strings.Contains(content, "fastapi"):
			s.Framework = "fastapi"
		case strings.Contains(content, "flask"):
			s.Framework = "flask"
		}
	}

	return s
}

func detectRust(root string) *Stack {
	data, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil {
		return nil
	}
	s := &Stack{Language: "rust"}
	content := string(data)

	switch {
	case strings.Contains(content, "actix-web"):
		s.Framework = "actix"
	case strings.Contains(content, "axum"):
		s.Framework = "axum"
	case strings.Contains(content, "rocket"):
		s.Framework = "rocket"
	case strings.Contains(content, "warp"):
		s.Framework = "warp"
	}

	return s
}

func detectKotlin(root string) *Stack {
	// Android: look for build.gradle.kts or build.gradle with kotlin
	for _, f := range []string{"build.gradle.kts", "app/build.gradle.kts"} {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "android") || strings.Contains(string(data), "kotlin") {
			return &Stack{Language: "kotlin", Framework: "android"}
		}
	}
	return nil
}

func detectSwift(root string) *Stack {
	// look for Package.swift or *.xcodeproj
	if _, err := os.Stat(filepath.Join(root, "Package.swift")); err == nil {
		return &Stack{Language: "swift"}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xcodeproj") || strings.HasSuffix(e.Name(), ".xcworkspace") {
			return &Stack{Language: "swift", Framework: "xcode"}
		}
	}
	return nil
}

func detectDart(root string) *Stack {
	if _, err := os.Stat(filepath.Join(root, "pubspec.yaml")); err != nil {
		return nil
	}
	return &Stack{Language: "dart", Framework: "flutter"}
}

func detectCI(root string) string {
	checks := []struct {
		path string
		name string
	}{
		{".github/workflows", "github-actions"},
		{".gitlab-ci.yml", "gitlab-ci"},
		{"Jenkinsfile", "jenkins"},
		{".circleci", "circleci"},
		{".travis.yml", "travis"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(root, c.path)); err == nil {
			return c.name
		}
	}
	return ""
}

func detectDatabase(root string) string {
	// scan common config files for database indicators
	scanFiles := []string{"docker-compose.yml", "docker-compose.yaml", ".env", ".env.example"}
	for _, f := range scanFiles {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		switch {
		case strings.Contains(content, "postgres"):
			return "postgres"
		case strings.Contains(content, "mongodb") || strings.Contains(content, "mongo:"):
			return "mongodb"
		case strings.Contains(content, "mysql"):
			return "mysql"
		case strings.Contains(content, "redis"):
			return "redis"
		case strings.Contains(content, "sqlite"):
			return "sqlite"
		}
	}
	return ""
}

func hasKey(m map[string]string, key string) bool {
	_, ok := m[key]
	return ok
}
