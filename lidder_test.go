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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Zuite struct {
	suite.Suite
	defs *defs
}

func (s *Zuite) SetupTest() {
	conf := `
include:
  - ^abc/.*\.go$
  - ^def/.*\.go$
exclude:
  - ^.*\bvendor/.*$
  - ^.*_test\.go$

rules:
  - name: some cool rule
    pattern: panic\(
    expected:
      - file_a.go
      - file_b.go`

	defs, err := parse([]byte(conf))
	if err != nil {
		panic(err)
	}

	s.defs = defs
}

func (s *Zuite) TestParseConfiguration() {
	require.Equal(s.T(), []string{"^abc/.*\\.go$", "^def/.*\\.go$"}, s.defs.Include)
	require.Equal(s.T(), []string{"^.*\\bvendor/.*$", "^.*_test\\.go$"}, s.defs.Exclude)
	require.Equal(s.T(), 1, len(s.defs.Rules))
	for _, rule := range s.defs.Rules {
		require.Equal(s.T(), "panic\\(", rule.Pattern)
		require.Equal(s.T(), []string{"file_a.go", "file_b.go"}, rule.Expected)
	}
}

func (s *Zuite) TestShouldCheck() {
	assert.False(s.T(), s.defs.shouldCheck("abc/hello_test.go"))
	assert.True(s.T(), s.defs.shouldCheck("abc/hello.go"))
	assert.True(s.T(), s.defs.shouldCheck("def/goodbye.go"))
	assert.False(s.T(), s.defs.shouldCheck("abcdef/goodbye.go"))
}

func (s *Zuite) TestRuleMatching() {
	testLines := []string{
		"if blah == blahblah {",
		"    panic(\"whoa\")",
		"}",
	}

	for _, rule := range s.defs.Rules {
		require.Equal(s.T(), 0, len(rule.actualFilenames))
	}

	for _, line := range testLines {
		s.defs.matchAgainstLine("file_c.go", line)
	}

	for _, rule := range s.defs.Rules {
		require.Equal(s.T(), map[string]bool{"file_c.go": true}, rule.actualFilenames)
		shouldNotBeThere, shouldBeThere := rule.Mismatches()
		require.Equal(s.T(), []string{"file_c.go"}, shouldNotBeThere)
		sort.Strings(shouldBeThere)
		require.Equal(s.T(), []string{"file_a.go", "file_b.go"}, shouldBeThere)
	}

	for _, line := range testLines {
		s.defs.matchAgainstLine("file_a.go", line)
	}

	for _, rule := range s.defs.Rules {
		require.Equal(s.T(), map[string]bool{"file_c.go": true, "file_a.go": true}, rule.actualFilenames)
		shouldNotBeThere, shouldBeThere := rule.Mismatches()
		require.Equal(s.T(), []string{"file_c.go"}, shouldNotBeThere)
		require.Equal(s.T(), []string{"file_b.go"}, shouldBeThere)
	}
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}
