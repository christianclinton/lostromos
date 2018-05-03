// Copyright 2017 the lostromos Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tmpl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	r := testCR.Name()
	assert.Equal(t, "dory", r)
}

func TestGetField(t *testing.T) {
	r := testCR.GetField("metadata", "name")
	assert.Equal(t, "dory", r)
}

func TestGetNumericField(t *testing.T) {
	r := testCR.GetField("spec", "Likes")
	assert.Equal(t, "32", r)
}

func TestGetFieldReturnsEmptyIfNotFound(t *testing.T) {
	r := testCR.GetField("Something", "made", "up")
	assert.Empty(t, r)
}
