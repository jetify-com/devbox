// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package services

type Services map[string]Service // name -> Service

type Service struct {
	Name               string
	ProcessComposePath string
}
