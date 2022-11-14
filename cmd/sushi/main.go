package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/yatoub/sushi"
)

const prev = "-parent-"

var (
	Build = "devel"
	V     = flag.Bool("version", false, "show version")
	H     = flag.Bool("help", false, "show help")
	S     = flag.Bool("s", false, "use local ssh config '~/.ssh/config'")

	log = sushi.GetLogger()

	templates = &promptui.SelectTemplates{
		Label:    "🖥 {{ . | green}}",
		Active:   "➤ {{ .Name | cyan  }}{{if .Alias}}({{.Alias | yellow}}){{end}} {{if .Host}}{{if .User}}{{.User | faint}}{{`@` | faint}}{{end}}{{.Host | faint}}{{end}}",
		Inactive: "  {{.Name | faint}}{{if .Alias}}({{.Alias | faint}}){{end}} {{if .Host}}{{if .User}}{{.User | faint}}{{`@` | faint}}{{end}}{{.Host | faint}}{{end}}",
	}
)

func findAlias(nodes []*sushi.Node, nodeAlias string) *sushi.Node {
	for _, node := range nodes {
		if node.Alias == nodeAlias {
			return node
		}
		if len(node.Children) > 0 {
			return findAlias(node.Children, nodeAlias)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		return
	}

	if *H {
		flag.Usage()
		return
	}

	if *V {
		fmt.Println("寿司 sushi - ssh user settings hosts import")
		fmt.Println("  git version:", Build)
		fmt.Println("  go version :", runtime.Version())
		return
	}
	if *S {
		err := sushi.LoadSshConfig()
		if err != nil {
			log.Error("load ssh config error", err)
			os.Exit(1)
		}
	} else {
		err := sushi.LoadConfig()
		if err != nil {
			log.Error("load config error", err)
			os.Exit(1)
		}
		fmt.Println("寿司 sushi - ssh user settings hosts import")
	}

	// login by alias
	if len(os.Args) > 1 {
		var nodeAlias = os.Args[1]
		var nodes = sushi.GetConfig()
		var node = findAlias(nodes, nodeAlias)
		if node != nil {
			client := sushi.NewClient(node)
			client.Login()
			return
		}
	}

	node := choose(nil, sushi.GetConfig())
	if node == nil {
		return
	}

	client := sushi.NewClient(node)
	client.Login()
}

func choose(parent, trees []*sushi.Node) *sushi.Node {
	prompt := promptui.Select{
		Label:        "select host or group",
		Items:        trees,
		Templates:    templates,
		Size:         20,
		HideSelected: true,
		Searcher: func(input string, index int) bool {
			searcher := func(node *sushi.Node, input string, index int) bool {
				content := fmt.Sprintf("%s %s %s", node.Name, node.User, node.Host)
				if strings.Contains(input, " ") {
					for _, key := range strings.Split(input, " ") {
						key = strings.TrimSpace(key)
						if key != "" {
							if !strings.Contains(content, key) {
								return false
							}
						}
					}
					return true
				}
				if strings.Contains(content, input) {
					return true
				}
				return false
			}
			node := trees[index]
			for _, children := range node.Children {
				if searcher(children, input, index) {
					return true
				}
			}
			return searcher(node, input, index)
		},
	}
	index, _, err := prompt.Run()
	if err != nil {
		return nil
	}

	node := trees[index]
	if len(node.Children) > 0 {
		first := node.Children[0]
		if first.Name != prev {
			// children
			// node.Children[0].Password
			for _, childNode := range node.Children {
				if childNode.Port == 0 && node.Port != 0 {
					childNode.Port = node.Port
				}
				if childNode.User == "" && node.User != "" {
					childNode.User = node.User
				}
				if childNode.Password == "" && node.Password != "" {
					childNode.Password = node.Password
				}
				if childNode.KeyPath == "" && node.KeyPath != "" {
					childNode.KeyPath = node.KeyPath
				}
				if childNode.Passphrase == "" && node.Passphrase != "" {
					childNode.Passphrase = node.Passphrase
				}
				if childNode.ProxyHost == "" && node.ProxyHost != "" {
					childNode.ProxyHost = node.ProxyHost
				}
			}
			first = &sushi.Node{Name: prev}
			node.Children = append(node.Children[:0], append([]*sushi.Node{first}, node.Children...)...)

		}
		return choose(trees, node.Children)
	}
	if node.Name == prev {
		if parent == nil {
			return choose(nil, sushi.GetConfig())
		}
		return choose(nil, parent)
	}

	return node
}
