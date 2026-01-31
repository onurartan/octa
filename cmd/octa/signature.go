package main

import (
	"fmt"
	"github.com/fatih/color"
)

func printSignature() {
	cyan := color.New(color.FgHiCyan, color.Bold).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()
	blueLink := color.New(color.FgHiBlue, color.Underline).SprintFunc()

	fmt.Println() 
	fmt.Printf("%s : %s\n", cyan("Author     "), white("Onur Artan"))
	fmt.Printf("%s : %s\n", cyan("Project    "), white("UnixId Octa"))
	fmt.Printf("%s : %s\n", cyan("Repository "), blueLink("github.com/onurartan/unixid-octa"))
	fmt.Println() 
}