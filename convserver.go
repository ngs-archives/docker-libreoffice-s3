package main

import (
	"fmt"
	"log"
	"os/exec"
)

func main() {
	go runCommand()
	fmt.Println("vim-go")
	for {
	}
}

func runCommand() {
	cmd := exec.Command("lowriter",
		"--invisible",
		"--convert-to",
		"pdf:writer_pdf_Export",
		"--outdir",
		"/var/files",
		"/var/files/test.pptx")

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Error starting: %v", err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatalf("Error starting: %v", err)
	}
	fmt.Println("I'm ok")
}
