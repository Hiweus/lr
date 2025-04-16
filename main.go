package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "lambda:run",
		Short: "Run a lambda in a docker container",
		RunE:  runLambda,
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runLambda(cmd *cobra.Command, args []string) error {
	events := getFilenames("events")
	var filenameEvent string
	if len(events) > 0 {
		prompt := promptui.Select{
			Label: "Filename event",
			Items: events,
		}
		_, result, err := prompt.Run()
		if err != nil {
			return err
		}
		filenameEvent = result
	}

	filesToIgnore := map[string]bool{
		"index.ts":         true,
		"lambda-runner.ts": true,
		"environment.d.ts": true,
		"cli.ts":           true,
		"bootstrap.ts":     true,
	}

	allFiles := getFilenames("src")
	var lambdas []string
	for _, file := range allFiles {
		if strings.HasSuffix(file, ".ts") && !filesToIgnore[file] {
			lambdas = append(lambdas, file)
		}
	}

	prompt := promptui.Select{
		Label: "Filename lambda",
		Items: lambdas,
	}
	_, filenameLambda, err := prompt.Run()
	if err != nil || filenameLambda == "" {
		fmt.Println("Lambda filename is required")
		return err
	}

	command := getDockerRunLambdaCommand("Dockerfile", filepath.Join("events", filenameEvent))
	fmt.Println("Running command:", strings.Join(command, " "))

	runnerTemplate, err := getLambdaRunnerTemplate("lambda-runner.template.ts", filenameLambda)
	if err != nil {
		return err
	}

	lambdaRunnerPath := filepath.Join("src", "lambda-runner.ts")
	if err := os.WriteFile(lambdaRunnerPath, []byte(runnerTemplate), 0644); err != nil {
		return err
	}

	defer os.Remove(lambdaRunnerPath)

	if err := execute(command, "."); err != nil {
		return err
	}

	return nil
}

func getFilenames(relativePath string) []string {
	absolutePath, err := filepath.Abs(relativePath)
	if err != nil {
		return []string{}
	}

	var filenames []string
	files, err := os.ReadDir(absolutePath)
	if err != nil {
		return []string{}
	}

	for _, file := range files {
		filenames = append(filenames, file.Name())
	}
	return filenames
}

func getDockerImageVersion(dockerfilePath string) string {
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return "node:20-bookworm-slim"
	}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToUpper(line), "FROM") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				return parts[1]
			}
		}
	}
	return "node:20-bookworm-slim"
}

func getDockerRunLambdaCommand(filenameDockerfile, filenameEvent string) []string {
	return []string{
		"docker", "run", "-it", "--network", "gdock_backend",
		"-v", fmt.Sprintf("%s:/app", "."), "-w", "/app",
		getDockerImageVersion(filenameDockerfile),
		"npx", "nodemon", "--exec", "ts-node -r tsconfig-paths/register src/lambda-runner.ts", filenameEvent,
	}
}

func execute(commandParts []string, context string) error {
	cmd := exec.Command(commandParts[0], commandParts[1:]...)
	cmd.Dir = context
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "TZ=UTC")
	return cmd.Run()
}

func getLambdaRunnerTemplate(_ string, lambdaName string) (string, error) {
	template := `
import { Context } from 'aws-lambda'
import { handler } from './{LAMBDA_NAME}'
import { readFileSync } from 'fs'

const filenameEvent = process.argv.at(2)
function loadEvent(filename: string | undefined): unknown {
  if(!filename) {
    return {}
  }

  const file = readFileSync(filename, 'utf-8')
  return JSON.parse(file)
}

async function main() {
  const event = loadEvent(filenameEvent)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  await handler(event as any, null as unknown as Context, () => {})
}

main().then(() => {
  // eslint-disable-next-line no-console
  console.log('done')
})
`
	nameWithoutExtension := strings.TrimSuffix(lambdaName, ".ts")
	return strings.ReplaceAll(template, "{LAMBDA_NAME}", nameWithoutExtension), nil
}
