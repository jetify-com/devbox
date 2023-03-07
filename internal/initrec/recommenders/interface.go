// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package recommenders

type Recommender interface {
	IsRelevant() bool
	Packages() []string
}
