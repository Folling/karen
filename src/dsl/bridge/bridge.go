/*
 *
 * Copyright (C) 2015-2017 Lukas Breuer. All rights reserved.
 *
 * This file is a part of the Karen Discord-Bot Project ("Karen").
 *
 * Karen is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License,
 * or (at your option) any later version.
 *
 * Karen is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 * See the GNU Affero General Public License for more details.
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package dsl_bridge

import (
    "sync"
)

var (
    scripts     = []Script{}
    scriptMutex = sync.RWMutex{}
)

func PushScript(v Script) {
    scriptMutex.Lock()
    defer scriptMutex.Unlock()

    scripts = append(scripts, v)
}

func GetScripts() (*[]Script) {
    scriptMutex.RLock()
    defer scriptMutex.RUnlock()

    return &scripts
}