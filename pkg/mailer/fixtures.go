package mailer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/model"
)

func (s Service) getTemplatePath(templateName string) string {
	return path.Join(s.templatesDir, templateName)
}

func (s Service) getFixturePath(templateName, fixtureName string) string {
	return path.Join(s.templatesDir, templateName, fmt.Sprintf("%s%s", fixtureName, jsonExtension))
}

func isExists(path string, directory bool) error {
	if info, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return model.ErrNotFound
		}
		return err
	} else if directory && !info.IsDir() {
		return model.ErrNotFound
	}

	return nil
}

func (s Service) ListFixtures(name string) ([]string, error) {
	templatePath := s.getTemplatePath(name)
	if err := isExists(templatePath, true); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(templatePath)
	if err != nil {
		return nil, fmt.Errorf("read templates directory: %w", err)
	}

	var fixtureList []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), jsonExtension) {
			fixtureList = append(fixtureList, strings.TrimSuffix(file.Name(), jsonExtension))
		}
	}

	return fixtureList, nil
}

func (s Service) GetFixture(name, fixture string) (map[string]any, error) {
	templatePath := s.getTemplatePath(name)
	if err := isExists(templatePath, true); err != nil {
		return nil, fmt.Errorf("is exists `%s`: %w", templatePath, err)
	}

	fixturePath := s.getFixturePath(name, fixture)
	if err := isExists(fixturePath, false); err != nil {
		return nil, fmt.Errorf("is exists `%s`: %w", fixturePath, err)
	}

	reader, err := os.OpenFile(fixturePath, os.O_RDONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open file `%s`: %w", fixturePath, err)
	}

	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			slog.LogAttrs(context.Background(), slog.LevelError, "close", slog.String("fn", "mailer.GetFixture"), slog.String("item", fixturePath), slog.Any("error", err))
		}
	}()

	var content map[string]any
	if err := json.NewDecoder(reader).Decode(&content); err != nil {
		return nil, fmt.Errorf("parse JSON fixture: %w", err)
	}

	return content, nil
}
