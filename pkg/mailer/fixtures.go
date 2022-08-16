package mailer

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
)

func (a App) getTemplatePath(templateName string) string {
	return path.Join(a.templatesDir, templateName)
}

func (a App) getFixturePath(templateName, fixtureName string) string {
	return path.Join(a.templatesDir, templateName, fmt.Sprintf("%s%s", fixtureName, jsonExtension))
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

// ListFixtures for a given template names
func (a App) ListFixtures(name string) ([]string, error) {
	templatePath := a.getTemplatePath(name)
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

// GetFixture for a template and a given name
func (a App) GetFixture(name, fixture string) (map[string]any, error) {
	templatePath := a.getTemplatePath(name)
	if err := isExists(templatePath, true); err != nil {
		return nil, err
	}

	fixturePath := a.getFixturePath(name, fixture)
	if err := isExists(fixturePath, false); err != nil {
		return nil, err
	}

	reader, err := os.OpenFile(fixturePath, os.O_RDONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open file `%s`: %w", fixturePath, err)
	}

	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			logger.WithField("fn", "mailer.GetFixture").WithField("item", fixturePath).Error("close: %s", err)
		}
	}()

	var content map[string]any
	if err := json.NewDecoder(reader).Decode(&content); err != nil {
		return nil, fmt.Errorf("parse JSON fixture: %w", err)
	}

	return content, nil
}
