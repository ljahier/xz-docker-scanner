package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Images []string `yaml:"images"`
}

type ImageCheckResult struct {
	ImageName string
	Output    string
	Error     error
}

func readConfig(configPath string) (*Config, error) {
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func pullImage(cli *client.Client, ctx context.Context, imageName string) error {
	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()

	fd, isTerminal := term.GetFdInfo(os.Stderr)
	if err := jsonmessage.DisplayJSONMessagesStream(out, os.Stderr, fd, isTerminal, nil); err != nil {
		return err
	}

	return nil
}

func checkXZInImage(image string, resultsChan chan<- ImageCheckResult, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		resultsChan <- ImageCheckResult{ImageName: image, Error: err}
		return
	}

	if err := pullImage(cli, ctx, image); err != nil {
		resultsChan <- ImageCheckResult{ImageName: image, Error: fmt.Errorf("error pulling image: %w", err)}
		return
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image,
		Cmd:   []string{"sh", "-c", "xz --version"},
	}, nil, nil, nil, "")
	if err != nil {
		resultsChan <- ImageCheckResult{ImageName: image, Error: err}
		return
	}
	containerID := resp.ID

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		resultsChan <- ImageCheckResult{ImageName: image, Error: err}
		return
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			resultsChan <- ImageCheckResult{ImageName: image, Error: err}
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		resultsChan <- ImageCheckResult{ImageName: image, Error: err}
		return
	}
	defer out.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, out)
	if err != nil {
		resultsChan <- ImageCheckResult{ImageName: image, Error: err}
		return
	}

	resultsChan <- ImageCheckResult{ImageName: image, Output: buf.String()}

	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		fmt.Printf("Error removing container %s: %v\n", containerID, err)
	}
}

func writeReport(results []ImageCheckResult, filename string) error {
	reportFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer reportFile.Close()

	for _, result := range results {
		var reportLine string
		cleanOutput := regexp.MustCompile(`[\x00-\x1F\x7F-\x9F]`).ReplaceAllString(result.Output, "")
		if result.Error != nil {
			reportLine = fmt.Sprintf("Image: %s - Error: %v\n", result.ImageName, result.Error)
		} else if strings.Contains(cleanOutput, "xz") {
			// Extract the first line or relevant part as the version info
			lines := strings.Split(cleanOutput, "\n")
			versionInfo := "version info not found"
			for _, line := range lines {
				if strings.Contains(line, "xz") {
					versionInfo = strings.TrimSpace(line)
					break
				}
			}
			reportLine = fmt.Sprintf("Image: %s - %s\n", result.ImageName, versionInfo)
		} else {
			reportLine = fmt.Sprintf("Image: %s - xz not found\n", result.ImageName)
		}
		_, err := reportFile.WriteString(reportLine)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	configPath := "images.yaml"
	config, err := readConfig(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	resultsChan := make(chan ImageCheckResult, len(config.Images))
	var results []ImageCheckResult

	for _, image := range config.Images {
		wg.Add(1)
		go checkXZInImage(image, resultsChan, &wg)
	}

	wg.Wait()
	close(resultsChan)

	for result := range resultsChan {
		results = append(results, result)
	}

	reportErr := writeReport(results, "report.txt")
	if reportErr != nil {
		fmt.Printf("Error writing report: %v\n", reportErr)
	}
}
