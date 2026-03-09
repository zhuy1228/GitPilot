package main

import (
	"fmt"

	"github.com/zhuy1228/GitPilot/internal/git"
)

func main() {
	gitClient := git.NewGitClient()
	changes, err := gitClient.StagedFiles("F:\\Project\\nakama")
	if err != nil {
		panic(err)
	}
	fmt.Printf("共 %d 个文件变更:\n", len(changes))
	for _, c := range changes {
		fmt.Println(c)
	}
}
