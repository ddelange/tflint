package tflint

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/spf13/afero"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func assertAppError(t *testing.T, expected Error, got error) {
	if appErr, ok := got.(*Error); ok {
		if appErr == nil {
			t.Fatalf("expected err is `%s`, but nothing occurred", expected.Error())
		}
		if appErr.Code != expected.Code {
			t.Fatalf("expected error code is `%d`, but get `%d`", expected.Code, appErr.Code)
		}
		if appErr.Level != expected.Level {
			t.Fatalf("expected error level is `%d`, but get `%d`", expected.Level, appErr.Level)
		}
		if appErr.Error() != expected.Error() {
			t.Fatalf("expected error is `%s`, but get `%s`", expected.Error(), appErr.Error())
		}
	} else {
		t.Fatalf("unexpected error occurred: %s", got)
	}
}

func testRunnerWithInputVariables(t *testing.T, files map[string]string, variables ...terraform.InputValues) *Runner {
	config := EmptyConfig()
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	for name, src := range files {
		err := fs.WriteFile(name, []byte(src), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	}

	loader, err := NewLoader(fs, config)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := loader.LoadConfig(".")
	if err != nil {
		t.Fatal(err)
	}

	runner, err := NewRunner(config, map[string]Annotations{}, cfg, variables...)
	if err != nil {
		t.Fatal(err)
	}

	return runner
}

func withEnvVars(t *testing.T, envVars map[string]string, test func()) {
	for key, value := range envVars {
		err := os.Setenv(key, value)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for key := range envVars {
			err := os.Unsetenv(key)
			if err != nil {
				t.Fatal(err)
			}
		}
	}()

	test()
}

func withinFixtureDir(t *testing.T, dir string, test func()) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(currentDir)

	err = os.Chdir(filepath.Join(currentDir, "test-fixtures", dir))
	if err != nil {
		t.Fatal(err)
	}

	test()
}

func testRunnerWithOsFs(t *testing.T, config *Config) *Runner {
	loader, err := NewLoader(afero.Afero{Fs: afero.NewOsFs()}, config)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := loader.LoadConfig(".")
	if err != nil {
		t.Fatal(err)
	}

	runner, err := NewRunner(config, map[string]Annotations{}, cfg, map[string]*terraform.InputValue{})
	if err != nil {
		t.Fatal(err)
	}

	return runner
}

func testRunnerWithAnnotations(t *testing.T, files map[string]string, annotations map[string]Annotations) *Runner {
	config := EmptyConfig()
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	for name, src := range files {
		err := fs.WriteFile(name, []byte(src), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	}

	loader, err := NewLoader(fs, config)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := loader.LoadConfig(".")
	if err != nil {
		t.Fatal(err)
	}

	runner, err := NewRunner(config, annotations, cfg, map[string]*terraform.InputValue{})
	if err != nil {
		t.Fatal(err)
	}

	return runner
}

func moduleConfig() *Config {
	c := EmptyConfig()
	c.Module = true
	return c
}

func newLine() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}
