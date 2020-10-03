package fixtures

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/httpjson"
)

const (
	templatesDir  = "./templates/"
	jsonExtension = ".json"
)

var (
	// ErrNoTemplate error occurs when template is not found
	ErrNoTemplate = errors.New("no template found")
)

func getTemplatePath(templateName string) string {
	return fmt.Sprintf("%s/%s/", templatesDir, templateName)
}

func getFixturePath(templateName, fixtureName string) string {
	return fmt.Sprintf("%s/%s/%s%s", templatesDir, templateName, fixtureName, jsonExtension)
}

func checkExist(path string, directory bool) error {
	if info, err := os.Stat(path); err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return ErrNoTemplate
		}
		return err
	} else if directory && !info.IsDir() {
		return ErrNoTemplate
	}

	return nil
}

func listHandler(w http.ResponseWriter, r *http.Request, templateName string) {
	templatePath := getTemplatePath(templateName)

	if err := checkExist(templatePath, true); err != nil {
		if err == ErrNoTemplate {
			httperror.NotFound(w)
			return
		}
		httperror.InternalServerError(w, err)
		return
	}

	files, err := ioutil.ReadDir(templatePath)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	fixtureList := make([]string, 0)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), jsonExtension) {
			fixtureList = append(fixtureList, strings.TrimSuffix(file.Name(), jsonExtension))
		}
	}

	httpjson.ResponseArrayJSON(w, http.StatusOK, fixtureList, httpjson.IsPretty(r))
}

// Get retrieves fixture content
func Get(templateName, fixtureName string) (map[string]interface{}, error) {
	templatePath := getTemplatePath(templateName)
	fixturePath := getFixturePath(templateName, fixtureName)

	if err := checkExist(templatePath, true); err != nil {
		return nil, err
	} else if err := checkExist(fixturePath, false); err != nil {
		return nil, err
	}

	rawContent, err := ioutil.ReadFile(fixturePath)
	if err != nil {
		return nil, err
	}

	var content map[string]interface{}
	if err := json.Unmarshal(rawContent, &content); err != nil {
		return nil, err
	}

	return content, nil
}

func getHandler(w http.ResponseWriter, r *http.Request, templateName, fixtureName string) {
	if content, err := Get(templateName, fixtureName); err != nil {
		if err == ErrNoTemplate {
			httperror.NotFound(w)
		} else {
			httperror.InternalServerError(w, err)
		}
	} else {
		httpjson.ResponseJSON(w, http.StatusOK, content, httpjson.IsPretty(r))
	}
}

// Handler for fixture request. Should be use with net/http
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := strings.Trim(r.URL.Path, "/")
		if query == "" {
			httperror.NotFound(w)
			return
		}

		urlParts := strings.Split(query, "/")

		if len(urlParts) == 1 {
			listHandler(w, r, urlParts[0])
		} else if len(urlParts) == 2 {
			getHandler(w, r, urlParts[0], urlParts[1])
		} else {
			httperror.NotFound(w)
		}
	})
}
