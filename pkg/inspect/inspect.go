package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || strings.HasPrefix(path, "./.git") || info.Size() == 0 {
			return nil
		}

		fmt.Printf(">>>>>>>>%s\n", path)

		gitLogCmd := fmt.Sprintf("git log --follow --find-renames=40%% --pretty=format:\"%%ad%%x0A%%h%%x0A%%an%%x20<%%ae>%%x0A%%s\" -- \"%s\" | head -n 4", path)
		gitLogOutput, err := exec.Command("bash", "-c", gitLogCmd).Output()
		if err != nil {
			return err
		}
		fmt.Println(string(gitLogOutput))

		emailCmd := fmt.Sprintf("git log --follow --find-renames=40%% --pretty=format:\"%%ae\" -- \"%s\" | head -n 1", path)
		email, err := exec.Command("bash", "-c", emailCmd).Output()
		if err != nil {
			return err
		}
		commitDatesCmd := fmt.Sprintf("git log --author=%s --pretty=format:\"%%ad\" | head -n 1", strings.TrimSpace(string(email)))
		commitDates, err := exec.Command("bash", "-c", commitDatesCmd).Output()
		if err != nil {
			return err
		}
		fmt.Printf("%s commits authored by email (first commit on %s)\n", strings.Count(string(commitDates), "\n"), strings.TrimSpace(string(commitDates)))

		fmt.Println("========binwalk")
		binwalkCmd := fmt.Sprintf("binwalk --disasm --signature --opcodes --extract --matryoshka --depth=32 --rm \"%s\"", path)
		exec.Command("bash", "-c", binwalkCmd).Run()

		fmt.Println("^^^^^^^^strings")
		stringsCmd := fmt.Sprintf("strings \"%s\" | grep -E \"^.*([a-z]{3,}|[A-Z]{3,}|[A-Z][a-z]{2,}).*$\" | sort | uniq -c | sort -nr | awk '{$1=\"\";print}' | sed 's/^.//' | head -n 25", path)
		stringsOutput, err := exec.Command("bash", "-c", stringsCmd).Output()
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(strings.NewReader(string(stringsOutput)))
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		fmt.Println("<<<<<<<<")

		return nil
	})

	if err != nil {
		fmt.Printf("error walking the path: %v\n", err)
		return
	}
}
