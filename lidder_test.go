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
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Zuite struct {
	suite.Suite
}

func configFile() (*defs, error) {
	conf := `
include:
  - ^abc/.*\.go$
  - ^def/.*\.go$
exclude:
  - ^.*\bvendor/.*$
  - ^.*_test\.go$

rules:
  - some cool rule:
    pattern: panic\(
    expected:
      - file_a.go
      - file_b.go`

	return parse([]byte(conf))
}

func (s *Zuite) TestParseConfiguration() {
	d, err := configFile()
	require.NoError(s.T(), err)

	require.Equal(s.T(), []string{"^abc/.*\\.go$","^def/.*\\.go$"}, d.Include)
	require.Equal(s.T(), []string{"^.*\\bvendor/.*$", "^.*_test\\.go$"}, d.Exclude)
	require.Equal(s.T(), 1, len(d.Rules))
	for _, rule := range d.Rules {
		require.Equal(s.T(), "panic\\(", rule.Pattern)
		require.Equal(s.T(), []string{"file_a.go", "file_b.go"}, rule.Expected)
	}
}

func (s *Zuite) TestShouldCheck() {
	d, err := configFile()
	require.NoError(s.T(), err)

	require.False(s.T(), d.shouldCheck("abc/hello_test.go"))
	require.True(s.T(), d.shouldCheck("abc/hello.go"))
	require.True(s.T(), d.shouldCheck("def/goodbye.go"))
	require.False(s.T(), d.shouldCheck("abcdef/goodbye.go"))
}

func (s *Zuite) TestMatchAgainstLine() {
	d, err := configFile()
	require.NoError(s.T(), err)

	testLines := []string{
		"if blah == blahblah {",
		"    panic(\"whoa\")",
		"}",
	}

	for _, rule := range d.Rules {
		require.Equal(s.T(), 0, len(rule.actualFilenames))
	}

	for _, line := range testLines {
		d.matchAgainstLine("file_c.go", line)
	}

	for _, rule := range d.Rules {
		require.Equal(s.T(), map[string]bool{"file_c.go": true}, rule.actualFilenames)
		shouldNotBeThere, shouldBeThere := rule.Mismatches()
		require.Equal(s.T(), []string{"file_c.go"}, shouldNotBeThere)
		sort.Strings(shouldBeThere)
		require.Equal(s.T(), []string{"file_a.go","file_b.go"}, shouldBeThere)
	}

	for _, line := range testLines {
		d.matchAgainstLine("file_a.go", line)
	}

	for _, rule := range d.Rules {
		require.Equal(s.T(), map[string]bool{"file_c.go": true, "file_a.go":true}, rule.actualFilenames)
		shouldNotBeThere, shouldBeThere := rule.Mismatches()
		require.Equal(s.T(), []string{"file_c.go"}, shouldNotBeThere)
		require.Equal(s.T(), []string{"file_b.go"}, shouldBeThere)
	}
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}
