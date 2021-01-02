package mailer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	rendererModel "github.com/ViBiOh/httputils/v3/pkg/renderer/model"
)

func (a app) getTemplatePath(templateName string) string {
	return path.Join(a.templatesDir, templateName)
}

func (a app) getFixturePath(templateName, fixtureName string) string {
	return path.Join(a.templatesDir, templateName, fmt.Sprintf("%s%s", fixtureName, jsonExtension))
}

func isExists(path string, directory bool) error {
	if info, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return rendererModel.ErrNotFound
		}
		return err
	} else if directory && !info.IsDir() {
		return rendererModel.ErrNotFound
	}

	return nil
}

func (a app) ListFixtures(name string) ([]string, error) {
	templatePath := a.getTemplatePath(name)
	if err := isExists(templatePath, true); err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(templatePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read templates directory: %s", err)
	}

	var fixtureList []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), jsonExtension) {
			fixtureList = append(fixtureList, strings.TrimSuffix(file.Name(), jsonExtension))
		}
	}

	return fixtureList, nil
}

func (a app) GetFixture(name, fixture string) (map[string]interface{}, error) {
	templatePath := a.getTemplatePath(name)
	if err := isExists(templatePath, true); err != nil {
		return nil, err
	}

	fixturePath := a.getFixturePath(name, fixture)
	if err := isExists(fixturePath, false); err != nil {
		return nil, err
	}

	payload, err := ioutil.ReadFile(fixturePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read file: %w", err)
	}

	var content map[string]interface{}
	if err := json.Unmarshal(payload, &content); err != nil {
		return nil, fmt.Errorf("unable to parse fixture: %w", err)
	}

	return content, nil
}
