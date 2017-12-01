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
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Zuite struct {
	suite.Suite
}

func (s *Zuite) TestParseConfiguration() {
	conf := `
include:
  - ^.*\.go$
exclude:
  - ^.*\bvendor/.*$

rules:
  - some cool rule:
    pattern: panic\(
    expected:
      - file_a.go
      - file_b.go`

	d, err := parse([]byte(conf))
	require.NoError(s.T(), err)

	require.Equal(s.T(), []string{"^.*\\.go$"}, d.Include)
	require.Equal(s.T(), []string{"^.*\\bvendor/.*$"}, d.Exclude)
	require.Equal(s.T(), 1, len(d.Rules))
	for _, rule := range d.Rules {
		require.Equal(s.T(), "panic\\(", rule.Pattern)
		require.Equal(s.T(), []string{"file_a.go", "file_b.go"}, rule.Expected)
	}
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}
