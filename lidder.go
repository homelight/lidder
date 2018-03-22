// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v2"
)

type defs struct {
	Include []string
	Exclude []string
	Rules   []*rule

	include []*regexp.Regexp
	exclude []*regexp.Regexp
}

type rule struct {
	Name     string
	Pattern  string
	Expected []string

	pattern           *regexp.Regexp
	expectedFilenames map[string]bool
	actualFilenames   map[string]bool
}

func (rule *rule) String() string {
	if rule.Name == "" {
		return fmt.Sprintf("Lidded pattern '%s'", rule.Pattern)
	}
	return fmt.Sprintf("Lidded rule '%s' (pattern '%s')", rule.Name, rule.Pattern)
}

func parse(input []byte) (*defs, error) {
	// yaml parse
	var defs defs
	err := yaml.Unmarshal([]byte(input), &defs)
	if err != nil {
		return nil, err
	}

	// compile all patterns: include, exclue, and all rules' pattern
	defs.include = make([]*regexp.Regexp, len(defs.Include))
	for i, expr := range defs.Include {
		pattern, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		defs.include[i] = pattern
	}

	defs.exclude = make([]*regexp.Regexp, len(defs.Exclude))
	for i, expr := range defs.Exclude {
		pattern, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		defs.exclude[i] = pattern
	}

	for _, rule := range defs.Rules {
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return nil, err
		}
		rule.pattern = pattern
	}

	// initialize all maps
	for _, rule := range defs.Rules {
		rule.expectedFilenames = make(map[string]bool)
		rule.actualFilenames = make(map[string]bool)
		for _, path := range rule.Expected {
			rule.expectedFilenames[path] = true
		}
	}

	return &defs, nil
}

// for single file mode, make it expect *only* the single file if it was expected
func (defs *defs) adjustExpectedFilenames(filename string) {
	for _, r := range defs.Rules {
		newExpectedFilenames := make(map[string]bool)
		if r.expectedFilenames[filename] {
			newExpectedFilenames[filename] = true
		}
		r.expectedFilenames = newExpectedFilenames
	}
}

func (rule *rule) Mismatches() ([]string, []string) {
	var (
		shouldNotBeThere = make([]string, 0)
		shouldBeThere    = make([]string, 0)
	)
	for actual := range rule.actualFilenames {
		if !rule.expectedFilenames[actual] {
			shouldNotBeThere = append(shouldNotBeThere, actual)
		}
	}
	for expected := range rule.expectedFilenames {
		if !rule.actualFilenames[expected] {
			shouldBeThere = append(shouldBeThere, expected)
		}
	}
	return shouldNotBeThere, shouldBeThere
}

func (defs *defs) matchAgainstLine(filename, line string) {
	// for every line, match against all (would be nice to use channels for that)
	for _, rule := range defs.Rules {
		if rule.pattern.MatchString(line) {
			rule.actualFilenames[filename] = true
		}
	}
}

func (defs *defs) matchAgainstFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		defs.matchAgainstLine(filename, line)
	}
}

func (defs *defs) exploreDir(dirname string) error {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return err
	}

	for _, fi := range files {
		filename := filepath.Join(dirname, fi.Name())
		switch mode := fi.Mode(); {
		case mode.IsDir():
			err := defs.exploreDir(filename)
			if err != nil {
				return err
			}
		case mode.IsRegular():
			if defs.shouldCheck(filename) {
				err := defs.matchAgainstFile(filename)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (defs *defs) shouldCheck(filename string) bool {
	// prioritize exclusions over inclusions
	// any exclusion means we do not process the file
	for _, exclude := range defs.exclude {
		if exclude.Match([]byte(filename)) {
			return false
		}
	}

	// any inclusion means we process the file
	for _, include := range defs.include {
		if include.Match([]byte(filename)) {
			return true
		}
	}

	return false
}

func fullScanMode(configFilename, baseDir string) bool {
	results := parseConfig(configFilename)

	err := results.exploreDir(baseDir)
	if err != nil {
		oops(err)
	}

	success := true
	for _, rule := range results.Rules {
		shouldNotBeThere, shouldBeThere := rule.Mismatches()
		if len(shouldNotBeThere) != 0 || len(shouldBeThere) != 0 {
			success = false
			fmt.Printf("%s\n", rule)
			if len(shouldNotBeThere) != 0 {
				fmt.Println("  didn't expect to find:")
				for _, s := range shouldNotBeThere {
					fmt.Print("   - ")
					fmt.Println(s)
				}
			}
			if len(shouldBeThere) != 0 {
				fmt.Println("  expected exceptions which were missing:")
				for _, s := range shouldBeThere {
					fmt.Print("   - ")
					fmt.Println(s)
				}
			}
		}
	}

	return success
}

func singleFileMode(configFilename, filename string) bool {
	results := parseConfig(configFilename)

	if !results.shouldCheck(filename) {
		return true
	}

	results.adjustExpectedFilenames(filename)
	err := results.matchAgainstFile(filename)
	if err != nil {
		oops(err)
	}

	success := true
	for _, rule := range results.Rules {
		shouldNotBeThere, shouldBeThere := rule.Mismatches()
		if len(shouldNotBeThere) != 0 || len(shouldBeThere) != 0 {
			success = false
			if len(shouldNotBeThere) != 0 {
				fmt.Printf("%s found\n", rule)
			} else if len(shouldBeThere) != 0 { // mutually exclusive for a single file
				fmt.Printf("%s expected but not found\n", rule)
			}
		}
	}

	return success
}

func parseConfig(filename string) *defs {
	config, err := ioutil.ReadFile(filename)
	if err != nil {
		oops(err)
	}

	results, err := parse(config)
	if err != nil {
		oops(err)
	}

	return results
}

func main() {
	var handler func(string, string) bool
	var modePath string
	if len(os.Args) == 2 {
		handler = fullScanMode
		modePath = "."
	} else if len(os.Args) == 3 {
		handler = singleFileMode
		modePath = os.Args[2]
	} else {
		fmt.Println("usage: lidder config.yaml [file]")
		fmt.Println("  file     Optional. When specified, lidder only checks the specified file.")
		os.Exit(1)
	}

	if ok := handler(os.Args[1], modePath); !ok {
		fmt.Print("\nlid test failed. sorry.\n")
		os.Exit(2)
	}

	fmt.Println("ok\tlid on all the things, nothing to see here.")
	os.Exit(0)
}

func oops(err error) {
	fmt.Fprintf(os.Stderr, "%s", err)
	os.Exit(1)
}
