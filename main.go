package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type exportFile struct {
	Sources []struct {
		Packages []struct {
			PackageIdentifier string `json:"PackageIdentifier"`
		} `json:"Packages"`
	} `json:"Sources"`
}

func winget(args ...string) *exec.Cmd {
	cmd := exec.Command("winget", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func listPackages() ([]string, error) {
	tmp := filepath.Join(os.TempDir(), "wingetprune-export.json")
	defer os.Remove(tmp)

	out, err := exec.Command("winget", "export", "-o", tmp, "--accept-source-agreements").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("winget export falhou: %v\n%s", err, out)
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}) // remove BOM, se houver

	var ex exportFile
	if err := json.Unmarshal(data, &ex); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var ids []string
	for _, s := range ex.Sources {
		for _, p := range s.Packages {
			id := p.PackageIdentifier
			if id != "" && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	sort.Slice(ids, func(i, j int) bool { return strings.ToLower(ids[i]) < strings.ToLower(ids[j]) })
	return ids, nil
}

// columnStarts devolve a posição inicial de cada coluna do cabeçalho
// (colunas são separadas por 2+ espaços).
func columnStarts(h []rune) []int {
	var starts []int
	for i, r := range h {
		if r == ' ' {
			continue
		}
		if i == 0 || (h[i-1] == ' ' && (i < 2 || h[i-2] == ' ')) {
			starts = append(starts, i)
		}
	}
	return starts
}

// listNames monta um mapa id -> nome a partir da saída tabular do `winget list`.
// Usa a linha de traços como âncora, então não depende do idioma do cabeçalho.
func listNames() map[string]string {
	names := map[string]string{}
	out, err := exec.Command("winget", "list", "--accept-source-agreements", "--disable-interactivity").Output()
	if err != nil {
		return names
	}

	lines := strings.Split(string(out), "\n")
	sep := -1
	for i, l := range lines {
		l = strings.TrimSpace(l[strings.LastIndex(l, "\r")+1:])
		if len(l) > 10 && strings.Trim(l, "-") == "" {
			sep = i
			break
		}
	}
	if sep < 1 {
		return names
	}

	hdr := lines[sep-1]
	hdr = hdr[strings.LastIndex(hdr, "\r")+1:]
	starts := columnStarts([]rune(hdr))
	if len(starts) < 3 {
		return names
	}
	idStart, verStart := starts[1], starts[2]

	for _, l := range lines[sep+1:] {
		r := []rune(strings.TrimRight(l, "\r"))
		if len(r) <= idStart {
			continue
		}
		end := verStart
		if end > len(r) {
			end = len(r)
		}
		name := strings.TrimSpace(string(r[:idStart]))
		id := strings.TrimSpace(string(r[idStart:end]))
		if id != "" && name != "" {
			names[id] = name
		}
	}
	return names
}

func main() {
	dryRun := flag.Bool("dry-run", false, "só mostra o que seria desinstalado")
	flag.Parse()

	ids, err := listPackages()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	names := listNames()
	label := func(id string) string {
		if n, ok := names[id]; ok {
			return n + " - " + id
		}
		return id
	}
	if len(ids) == 0 {
		fmt.Println("Nenhum pacote encontrado.")
		return
	}

	in := bufio.NewReader(os.Stdin)
	var toRemove []string

	fmt.Printf("%d pacotes. [Enter]=manter  [u]=desinstalar  [i]=info  [q]=parar\n\n", len(ids))

loop:
	for idx := 0; idx < len(ids); idx++ {
		id := ids[idx]
		for {
			fmt.Printf("(%d/%d) %s > ", idx+1, len(ids), label(id))
			line, _ := in.ReadString('\n')
			switch strings.ToLower(strings.TrimSpace(line)) {
			case "", "k", "n":
				continue loop
			case "u", "d":
				toRemove = append(toRemove, id)
				continue loop
			case "i":
				_ = winget("show", "--id", id, "-e").Run()
			case "q":
				break loop
			default:
				fmt.Println("  use Enter, u, i ou q")
			}
		}
	}

	if len(toRemove) == 0 {
		fmt.Println("\nNada marcado para remover.")
		return
	}

	fmt.Println("\nMarcados para desinstalar:")
	for _, id := range toRemove {
		fmt.Println("  -", label(id))
	}

	if *dryRun {
		fmt.Println("\n(dry-run: nada foi desinstalado)")
		return
	}

	fmt.Print("\nConfirmar? [s/N] > ")
	ans, _ := in.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(ans)) != "s" {
		fmt.Println("Cancelado.")
		return
	}

	for _, id := range toRemove {
		fmt.Printf("\n==> Desinstalando %s\n", id)
		if err := winget("uninstall", "--id", id, "-e", "--accept-source-agreements").Run(); err != nil {
			fmt.Printf("    falhou: %v\n", err)
		}
	}
}
