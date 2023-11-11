package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/nsf/termbox-go"
)

const statusLineHeight = 1

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: nte <filename>")
		os.Exit(1)
	}

	filename := os.Args[1]

	content, err := readFile(filename)
	if os.IsNotExist(err) {
		fmt.Printf("File %s doesn't exist. Creating a new file.\n", filename)
		content = []string{""}
		err := writeFile(filename, content)
		if err != nil {
			fmt.Printf("Error creating file: %v\n", err)
			os.Exit(1)
		}
	} else if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	editor := Editor{
		filename: filename,
		content:  content,
	}

	if err := editor.run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

type Editor struct {
	filename string
	content  []string
	cursor   struct {
		x, y int
	}
}

func (e *Editor) display(startLine, visibleLines int) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	defer termbox.Flush()

	printString(0, 0, fmt.Sprintf("Editing: %s", e.filename), termbox.ColorWhite, termbox.ColorDefault)

	endLine := startLine + visibleLines
	if endLine > len(e.content) {
		endLine = len(e.content)
	}

	for i := startLine; i < endLine; i++ {
		printString(0, i-startLine+statusLineHeight, e.content[i], termbox.ColorDefault, termbox.ColorDefault)
	}

	termbox.SetCursor(e.cursor.x, e.cursor.y-startLine+statusLineHeight)
}

func (e *Editor) save() error {
	file, err := os.Create(e.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range e.content {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Editor) run() error {
	err := termbox.Init()
	if err != nil {
		return err
	}
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc)

	e.cursor.y = 0
	_, height := termbox.Size()
	visibleLines := height - statusLineHeight
	startLine := 0

	if len(e.content) == 0 {
		e.content = append(e.content, "")
	}

mainLoop:
	for {
		e.display(startLine, visibleLines)

		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyCtrlS:
				err := e.save()
				if err != nil {
					return err
				}
			case termbox.KeyCtrlQ:
				break mainLoop
			default:
				switch ev.Ch {
				case 'q':
					e.display(startLine, visibleLines)
					printString(0, len(e.content)+statusLineHeight, "Press q again to confirm exit", termbox.ColorRed, termbox.ColorDefault)
					termbox.Flush()
					ev2 := termbox.PollEvent()
					if ev2.Type == termbox.EventKey && ev2.Ch == 'q' {
						break mainLoop
					}
				}
			}

			if ev.Ch != 0 {
				if ev.Ch == '\n' {
					e.content = append(e.content[:e.cursor.y+1], "")
					copy(e.content[e.cursor.y+2:], e.content[e.cursor.y+1:])
					e.content[e.cursor.y+1] = e.content[e.cursor.y][e.cursor.x:]
					e.content[e.cursor.y] = e.content[e.cursor.y][:e.cursor.x]
					e.cursor.y++
					e.cursor.x = 0
				} else {
					e.content[e.cursor.y] = e.content[e.cursor.y][:e.cursor.x] + string(ev.Ch) + e.content[e.cursor.y][e.cursor.x:]
					e.cursor.x++
				}
			} else {
				switch ev.Key {
				case termbox.KeyBackspace, termbox.KeyBackspace2:
					if e.cursor.x > 0 {
						e.content[e.cursor.y] = e.content[e.cursor.y][:e.cursor.x-1] + e.content[e.cursor.y][e.cursor.x:]
						e.cursor.x--
					} else if e.cursor.y > 0 {
						prevLineLen := len(e.content[e.cursor.y-1])
						e.content[e.cursor.y-1] += e.content[e.cursor.y]
						e.content = append(e.content[:e.cursor.y], e.content[e.cursor.y+1:]...)
						e.cursor.y--
						e.cursor.x = prevLineLen
					}
				case termbox.KeyEnter:
					currentLine := e.content[e.cursor.y]
					newLine := currentLine[e.cursor.x:]
					e.content[e.cursor.y] = currentLine[:e.cursor.x]
					e.content = append(e.content[:e.cursor.y+1], append([]string{newLine}, e.content[e.cursor.y+1:]...)...)
					e.cursor.y++
					e.cursor.x = 0
				case termbox.KeyTab:
					e.content[e.cursor.y] = e.content[e.cursor.y][:e.cursor.x] + "\t" + e.content[e.cursor.y][e.cursor.x:]
					e.cursor.x++
				case termbox.KeySpace:
					e.content[e.cursor.y] = e.content[e.cursor.y][:e.cursor.x] + " " + e.content[e.cursor.y][e.cursor.x:]
					e.cursor.x++
				}
			}

			switch ev.Key {
			case termbox.KeyArrowLeft:
				if e.cursor.x > 0 {
					e.cursor.x--
				} else {
					if e.cursor.y > 0 {
						e.cursor.y--
						e.cursor.x = len(e.content[e.cursor.y])
					}
				}

			case termbox.KeyArrowRight:
				if e.cursor.x < len(e.content[e.cursor.y]) {
					e.cursor.x++
				} else {
					if e.cursor.y < len(e.content)-1 {
						e.cursor.y++
						e.cursor.x = 0
					}
				}
			case termbox.KeyArrowUp:
				if e.cursor.y > 0 {
					e.cursor.y--
					if startLine > 0 && e.cursor.y < startLine {
						startLine--
					}
				} else if startLine > 0 {
					startLine--
				}
			case termbox.KeyArrowDown:
				if e.cursor.y < len(e.content)-1 {
					e.cursor.y++
					if e.cursor.y >= startLine+visibleLines {
						startLine++
					}
				} else if startLine < len(e.content)-visibleLines {
					startLine++
				}
			}
		case termbox.EventError:
			return ev.Err
		}
	}

	return nil
}

func printString(x, y int, msg string, fg, bg termbox.Attribute) {
	for i, c := range msg {
		termbox.SetCell(x+i, y, c, fg, bg)
	}
}

func readFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var content []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content = append(content, scanner.Text())
	}

	return content, scanner.Err()
}

func writeFile(filename string, content []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range content {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}
